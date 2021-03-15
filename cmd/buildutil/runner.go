package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/goreleaser/nfpm"
	_ "github.com/goreleaser/nfpm/deb" // blank import to register the format
	_ "github.com/goreleaser/nfpm/rpm" // blank import to register the format
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/executil"
)

func call(ctx context.Context, command string, args ...string) {
	logrus.Debugf("$ %s %s", command, strings.Join(args, " "))
	c := exec.Command(command, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	cmdutil.Must(executil.Run(ctx, c))
}

type BuildParameters struct {
	TargetSystems  []string
	TargetPackages []string
	S3URL          string

	CreateCompressed bool
	CreateRPM        bool
	CreateDEB        bool

	CGO bool
}

type Runner struct {
	Info       BuildInfo
	Inst       *Instrumentation
	Parameters BuildParameters
}

func (r *Runner) Bind(cmd *cobra.Command) error {
	r.Inst = NewInstrumentation()

	cmd.PersistentFlags().StringSliceVarP(
		&r.Parameters.TargetSystems, "cross-compile", "x", []string{},
		"Targets for cross compilation (eg linux/amd64). Can be used multiple times.")
	cmd.PersistentFlags().StringSliceVarP(
		&r.Parameters.TargetPackages, "package", "p", []string{},
		"Packages to build.")
	cmd.PersistentFlags().StringVar(
		&r.Parameters.S3URL, "s3-url", "",
		"S3 URL to upload compiled releases.")

	cmd.PersistentFlags().BoolVar(
		&r.Parameters.CreateCompressed, "compress", false,
		"Creates .tgz artifacts for POSIX targets and .zip for windows.")
	cmd.PersistentFlags().BoolVar(
		&r.Parameters.CreateRPM, "rpm", false,
		"Creates .rpm artifacts for linux targets.")
	cmd.PersistentFlags().BoolVar(
		&r.Parameters.CreateDEB, "deb", false,
		"Creates .deb artifacts for linux targets.")
	cmd.PersistentFlags().BoolVar(
		&r.Parameters.CGO, "cgo", false,
		"Enable CGO.")

	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		defer r.Inst.Durations.Steps.Stopwatch("info")()
		info, err := CollectBuildInformation(context.Background(), r.Parameters)
		cmdutil.Must(err)

		dumpJSON(info)
		if len(info.Commit.DirtyFiles) > 0 {
			logrus.Warn("The repository contains uncommitted files!")
		}

		r.Info = info
	}

	cmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		dumpJSON(r.Inst)
	}

	return nil
}

func (r *Runner) dist(parts ...string) string {
	parts = append([]string{r.Info.Go.Dir, "dist"}, parts...)
	return path.Join(parts...)
}

func (r *Runner) RunAll(ctx context.Context, cmd *cobra.Command, args []string) {
	defer r.Inst.Durations.Steps.Stopwatch("all")()

	r.RunVendor(ctx, cmd, args)
	r.RunTest(ctx, cmd, args)
	r.RunBuild(ctx, cmd, args)
	r.RunArtifacts(ctx, cmd, args)
	r.RunUpload(ctx, cmd, args)
}

func (r *Runner) RunClean(ctx context.Context, cmd *cobra.Command, args []string) {
	defer r.Inst.Durations.Steps.Stopwatch("clean")()

	files, err := filepath.Glob(r.dist("*"))
	cmdutil.Must(err)

	for _, file := range files {
		logrus.Info("remove ", file)
		os.Remove(file)
	}
}

func (r *Runner) RunVendor(ctx context.Context, cmd *cobra.Command, args []string) {
	defer r.Inst.Durations.Steps.Stopwatch("vendor")()

	call(ctx, "go", "mod", "vendor")
}

func (r *Runner) RunTest(ctx context.Context, cmd *cobra.Command, args []string) {
	defer r.Inst.Durations.Steps.Stopwatch("test")()

	r.RunTestFormat(ctx, cmd, args)
	r.RunTestVet(ctx, cmd, args)
	r.RunTestPackages(ctx, cmd, args)
}

func (r *Runner) RunTestFormat(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"-s", "-l"}
	a = append(a, r.Info.Test.Files...)

	logrus.Info("Testing file formatting (gofmt)")
	defer r.Inst.Durations.Testing.Stopwatch("fmt")()
	call(ctx, "gofmt", a...)
}

func (r *Runner) RunTestVet(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"vet"}
	a = append(a, r.Info.Test.Packages...)

	logrus.Info("Testing suspicious constructs (go vet)")
	defer r.Inst.Durations.Testing.Stopwatch("vet")()
	call(ctx, "go", a...)
}

func (r *Runner) RunTestPackages(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"test"}
	a = append(a, r.Info.Test.Packages...)

	logrus.Info("Testing packages")
	defer r.Inst.Durations.Testing.Stopwatch("packages")()
	call(ctx, "go", a...)
}

func (r *Runner) RunBuild(ctx context.Context, cmd *cobra.Command, args []string) {
	defer r.Inst.Durations.Steps.Stopwatch("build")()

	for _, target := range r.Info.Targets {
		logrus.Infof("Building %s for %s", target.Package, target.System.Name())

		ldData := []struct {
			name  string
			value string
		}{
			{name: "Name", value: target.Name},
			{name: "Version", value: r.Info.Version.String()},
			{name: "GoModule", value: r.Info.Go.Module},
			{name: "GoPackage", value: target.Package},
			{name: "GoVersion", value: r.Info.Go.Version},
			{name: "BuildDate", value: r.Info.BuildDate},
			{name: "CommitDate", value: r.Info.Commit.Date},
			{name: "CommitHash", value: r.Info.Commit.Hash},
		}

		ldFlags := []string{}
		for _, entry := range ldData {
			ldFlags = append(ldFlags, fmt.Sprintf(
				`-X 'github.com/rebuy-de/rebuy-go-sdk/v3/pkg/cmdutil.%s=%s'`,
				entry.name, entry.value,
			))
		}

		cmdutil.Must(os.Setenv("GOOS", target.System.OS))
		cmdutil.Must(os.Setenv("GOARCH", target.System.Arch))
		if target.CGO {
			cmdutil.Must(os.Setenv("CGO_ENABLED", "1"))
		} else {
			cmdutil.Must(os.Setenv("CGO_ENABLED", "0"))
		}

		sw := r.Inst.Durations.Building.Stopwatch(target.Outfile)
		call(ctx, "go", "build",
			"-o", r.dist(target.Outfile),
			"-ldflags", "-s -w "+strings.Join(ldFlags, " "),
			target.Package)

		r.Inst.ReadSize(target.Outfile)

		sw()
	}
}

func (r *Runner) RunArtifacts(ctx context.Context, cmd *cobra.Command, args []string) {
	defer r.Inst.Durations.Steps.Stopwatch("artifacts")()

	for _, artifact := range r.Info.Artifacts {
		logrus.Infof("Creating artifact for %s", artifact.Filename)
		sw := r.Inst.Durations.Artifacts.Stopwatch(artifact.Filename)

		binaries := map[string]string{}
		for _, target := range r.Info.Targets {
			if target.System != artifact.System {
				continue
			}

			binaries[target.Name] = r.dist(target.Outfile)
		}

		switch artifact.Kind {
		default:
			logrus.Warnf("Unknown artifact kind %s", artifact.Kind)

		case "binary":
			// nothing to do here

		case "tgz":
			dst, err := os.Create(r.dist(artifact.Filename))
			cmdutil.Must(err)
			defer dst.Close()

			zw := gzip.NewWriter(dst)
			defer zw.Close()

			tw := tar.NewWriter(zw)
			defer tw.Close()

			for name, src := range binaries {
				f, err := os.Open(src)
				cmdutil.Must(err)
				defer f.Close()

				fi, err := f.Stat()
				cmdutil.Must(err)

				hdr := &tar.Header{
					Name: name,
					Mode: 0755,
					Size: fi.Size(),
				}
				err = tw.WriteHeader(hdr)
				cmdutil.Must(err)

				_, err = io.Copy(tw, f)
				cmdutil.Must(err)
			}

			cmdutil.Must(tw.Close())

		case "rpm":
			fallthrough

		case "deb":
			version, release := r.Info.Version.StringRelease()

			bindir := "/usr/share/bin"
			files := map[string]string{}
			for name, src := range binaries {
				files[src] = path.Join(bindir, name)
			}

			info := &nfpm.Info{
				Name:       r.Info.Go.Name,
				Arch:       artifact.System.Arch,
				Platform:   artifact.System.OS,
				Version:    version,
				Release:    release,
				Maintainer: "reBuy Platform Team <dl-scb-tech-platform@rebuy.com>",
				Bindir:     bindir,
				Overridables: nfpm.Overridables{
					Files: files,
				},
			}

			cmdutil.Must(nfpm.Validate(info))

			packager, err := nfpm.Get(artifact.Kind)
			cmdutil.Must(err)

			w, err := os.Create(r.dist(artifact.Filename))
			cmdutil.Must(err)
			defer w.Close()

			cmdutil.Must(packager.Package(nfpm.WithDefaults(info), w))
			cmdutil.Must(w.Close())
		}

		r.Inst.ReadSize(artifact.Filename)

		// create symlinks
		for _, link := range artifact.Aliases {
			fullLink := r.dist(link)
			os.Remove(fullLink)
			cmdutil.Must(os.Symlink(artifact.Filename, fullLink))
		}

		sw()
	}
}

func (r *Runner) RunUpload(ctx context.Context, cmd *cobra.Command, args []string) {
	defer r.Inst.Durations.Steps.Stopwatch("upload")()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithDefaultRegion("eu-west-1"))
	cmdutil.Must(err)

	uploader := manager.NewUploader(s3.NewFromConfig(cfg))
	for _, artifact := range r.Info.Artifacts {
		if artifact.S3Location == nil {
			continue
		}

		us := artifact.S3Location.String()
		logrus.Infof("Uploading %s", us)
		sw := r.Inst.Durations.Upload.Stopwatch(us)

		f, err := os.Open(r.dist(artifact.Filename))
		cmdutil.Must(err)

		tags := url.Values{}
		tags.Set("GoModule", r.Info.Go.Module)
		tags.Set("Branch", r.Info.Commit.Branch)
		tags.Set("System", artifact.System.Name())
		tags.Set("ReleaseKind", r.Info.Version.Kind)

		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket:  &artifact.S3Location.Bucket,
			Key:     &artifact.S3Location.Key,
			Tagging: aws.String(tags.Encode()),
			Body:    f,
		})
		cmdutil.Must(err)

		sw()
	}
}
