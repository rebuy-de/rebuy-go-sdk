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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/goreleaser/nfpm"
	_ "github.com/goreleaser/nfpm/deb" // blank import to register the format
	_ "github.com/goreleaser/nfpm/rpm" // blank import to register the format
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/executil"
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
}

type Runner struct {
	Info       BuildInfo
	Parameters BuildParameters
}

func (r *Runner) Bind(cmd *cobra.Command) error {
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

	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		info, err := CollectBuildInformation(context.Background(), r.Parameters)
		cmdutil.Must(err)

		dumpJSON(info)
		if len(info.Commit.DirtyFiles) > 0 {
			logrus.Warn("The repository contains uncommitted files!")
		}

		r.Info = info
	}

	return nil
}

func (r *Runner) dist(parts ...string) string {
	parts = append([]string{r.Info.Go.Dir, "dist"}, parts...)
	return path.Join(parts...)
}

func (r *Runner) RunAll(ctx context.Context, cmd *cobra.Command, args []string) {
	r.RunVendor(ctx, cmd, args)
	r.RunTest(ctx, cmd, args)
	r.RunBuild(ctx, cmd, args)
	r.RunArtifacts(ctx, cmd, args)
	r.RunUpload(ctx, cmd, args)
}

func (r *Runner) RunClean(ctx context.Context, cmd *cobra.Command, args []string) {
	files, err := filepath.Glob(r.dist("*"))
	cmdutil.Must(err)

	for _, file := range files {
		logrus.Info("remove ", file)
		os.Remove(file)
	}
}

func (r *Runner) RunVendor(ctx context.Context, cmd *cobra.Command, args []string) {
	call(ctx, "go", "mod", "vendor")
}

func (r *Runner) RunTest(ctx context.Context, cmd *cobra.Command, args []string) {
	r.RunTestFormat(ctx, cmd, args)
	r.RunTestVet(ctx, cmd, args)
	r.RunTestPackages(ctx, cmd, args)
}

func (r *Runner) RunTestFormat(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"-s", "-l"}
	a = append(a, r.Info.Test.Files...)

	logrus.Info("Testing file formatting (gofmt)")
	start := time.Now()
	call(ctx, "gofmt", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (r *Runner) RunTestVet(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"vet"}
	a = append(a, r.Info.Test.Packages...)

	logrus.Info("Testing suspicious constructs (go vet)")
	start := time.Now()
	call(ctx, "go", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (r *Runner) RunTestPackages(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"test"}
	a = append(a, r.Info.Test.Packages...)

	logrus.Info("Testing packages")
	start := time.Now()
	call(ctx, "go", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (r *Runner) RunBuild(ctx context.Context, cmd *cobra.Command, args []string) {
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
			{name: "BuildDate", value: r.Info.BuildDate},
			{name: "CommitDate", value: r.Info.Commit.Date},
			{name: "CommitHash", value: r.Info.Commit.Hash},
		}

		ldFlags := []string{}
		for _, entry := range ldData {
			ldFlags = append(ldFlags, fmt.Sprintf(
				`-X 'github.com/rebuy-de/rebuy-go-sdk/v2/pkg/cmdutil.%s=%s'`,
				entry.name, entry.value,
			))
		}

		os.Setenv("GOOS", target.System.OS)
		os.Setenv("GOARCH", target.System.Arch)
		os.Setenv("CGO_ENABLED", "0")

		start := time.Now()
		call(ctx, "go", "build",
			"-o", r.dist(target.Outfile),
			"-ldflags", "-s -w "+strings.Join(ldFlags, " "),
			target.Package)

		stat, err := os.Stat(r.dist(target.Outfile))
		cmdutil.Must(err)

		logrus.Infof("Build finished in %v with a size of %s",
			time.Since(start).Truncate(10*time.Millisecond),
			byteFormat(stat.Size()))
	}
}

func (r *Runner) RunArtifacts(ctx context.Context, cmd *cobra.Command, args []string) {
	for _, artifact := range r.Info.Artifacts {
		logrus.Infof("Creating artifact for %s", artifact.Filename)

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

		// create symlinks
		for _, link := range artifact.Aliases {
			fullLink := r.dist(link)
			os.Remove(fullLink)
			cmdutil.Must(os.Symlink(artifact.Filename, fullLink))
		}
	}
}

func (r *Runner) RunUpload(ctx context.Context, cmd *cobra.Command, args []string) {
	if r.Parameters.S3URL == "" {
		logrus.Warn("No S3 Bucket specified. Skipping upload.")
		return
	}

	s3url, err := url.Parse(r.Parameters.S3URL)
	cmdutil.Must(err)

	if s3url.Scheme != "s3" && s3url.Scheme != "" {
		cmdutil.Must(fmt.Errorf("Unknown scheme %s for the S3 URL", s3url.Scheme))
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	cmdutil.Must(err)

	uploader := s3manager.NewUploader(sess)
	for _, target := range r.Info.Targets {
		logrus.Infof("Uploading %s to s3://%s%s", target.Outfile, s3url.Host, s3url.Path)

		f, err := os.Open(r.dist(target.Outfile))
		cmdutil.Must(err)

		tags := url.Values{}
		tags.Set("GoModule", r.Info.Go.Module)
		tags.Set("GoPackage", target.Package)
		tags.Set("Branch", r.Info.Commit.Branch)
		tags.Set("System", target.System.Name())
		tags.Set("ReleaseKind", r.Info.Version.Kind)

		start := time.Now()
		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket:  &s3url.Host,
			Key:     aws.String(path.Join(s3url.Path, target.Outfile)),
			Tagging: aws.String(tags.Encode()),
			Body:    f,
		})
		cmdutil.Must(err)

		logrus.Infof("Upload finished in %v", time.Since(start).Truncate(10*time.Millisecond))
	}
}
