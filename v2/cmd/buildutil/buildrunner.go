package main

import (
	"context"
	"fmt"
	"io/ioutil"
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

type BuildRunner struct {
	Info       BuildInfo
	Parameters struct {
		TargetSystems  []string
		TargetPackages []string
		S3URL          string

		GeneratorTargetVersion string // for generate command
	}
}

func (r *BuildRunner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSliceVarP(
		&r.Parameters.TargetSystems, "cross-compile", "x", []string{},
		"Targets for cross compilation (eg linux/amd64). Can be used multiple times.")
	cmd.PersistentFlags().StringSliceVarP(
		&r.Parameters.TargetPackages, "package", "p", []string{},
		"Packages to build.")
	cmd.PersistentFlags().StringVar(
		&r.Parameters.S3URL, "s3-url", "",
		"S3 URL to upload compiled releases.")
	cmd.PersistentFlags().StringVar(
		&r.Parameters.GeneratorTargetVersion, "generator.target-version", "",
		"Target version for the generated ./buildutilw file.")

	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		info, err := CollectBuildInformation(context.Background(),
			r.Parameters.TargetPackages,
			r.Parameters.TargetSystems,
		)
		cmdutil.Must(err)

		dumpJSON(info)
		if len(info.Commit.DirtyFiles) > 0 {
			logrus.Warn("The repository contains uncommitted files!")
		}

		r.Info = info
	}

	return nil
}

func (r *BuildRunner) RunAll(ctx context.Context, cmd *cobra.Command, args []string) {
	r.RunVendor(ctx, cmd, args)
	r.RunTest(ctx, cmd, args)
	r.RunBuild(ctx, cmd, args)
	r.RunUpload(ctx, cmd, args)
}

func (r *BuildRunner) RunClean(ctx context.Context, cmd *cobra.Command, args []string) {
	files, err := filepath.Glob("dist/*")
	cmdutil.Must(err)

	for _, file := range files {
		logrus.Info("remove ", file)
		os.Remove(file)
	}
}

func (r *BuildRunner) RunVendor(ctx context.Context, cmd *cobra.Command, args []string) {
	call(ctx, "go", "mod", "vendor")
}

func (r *BuildRunner) RunTest(ctx context.Context, cmd *cobra.Command, args []string) {
	r.RunTestFormat(ctx, cmd, args)
	r.RunTestVet(ctx, cmd, args)
	r.RunTestPackages(ctx, cmd, args)
}

func (r *BuildRunner) RunTestFormat(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"-s", "-l"}
	a = append(a, r.Info.Test.Files...)

	logrus.Info("Testing file formatting (gofmt)")
	start := time.Now()
	call(ctx, "gofmt", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (r *BuildRunner) RunTestVet(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"vet"}
	a = append(a, r.Info.Test.Packages...)

	logrus.Info("Testing suspicious constructs (go vet)")
	start := time.Now()
	call(ctx, "go", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (r *BuildRunner) RunTestPackages(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"test"}
	a = append(a, r.Info.Test.Packages...)

	logrus.Info("Testing packages")
	start := time.Now()
	call(ctx, "go", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (r *BuildRunner) RunBuild(ctx context.Context, cmd *cobra.Command, args []string) {
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

		dist := path.Join(r.Info.Go.Dir, "dist")

		start := time.Now()
		call(ctx, "go", "build",
			"-o", path.Join(dist, target.Outfile.Name),
			"-ldflags", "-s -w "+strings.Join(ldFlags, " "),
			target.Package)

		for _, link := range target.Outfile.Aliases {
			fullLink := path.Join(dist, link)
			os.Remove(fullLink)
			cmdutil.Must(os.Symlink(target.Outfile.Name, fullLink))
		}

		stat, err := os.Stat(path.Join(dist, target.Outfile.Name))
		cmdutil.Must(err)

		logrus.Infof("Build finished in %v with a size of %s",
			time.Since(start).Truncate(10*time.Millisecond),
			byteFormat(stat.Size()))
	}
}

func (r *BuildRunner) RunUpload(ctx context.Context, cmd *cobra.Command, args []string) {
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
	dist := path.Join(r.Info.Go.Dir, "dist")

	for _, target := range r.Info.Targets {
		logrus.Infof("Uploading %s to s3://%s%s", target.Outfile.Name, s3url.Host, s3url.Path)

		f, err := os.Open(path.Join(dist, target.Outfile.Name))
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
			Key:     aws.String(path.Join(s3url.Path, target.Outfile.Name)),
			Tagging: aws.String(tags.Encode()),
			Body:    f,
		})
		cmdutil.Must(err)

		logrus.Infof("Upload finished in %v", time.Since(start).Truncate(10*time.Millisecond))
	}
}

func (r *BuildRunner) RunGenerateWrapper(ctx context.Context, cmd *cobra.Command, args []string) {
	contents, err := generateWrapper(r.Parameters.GeneratorTargetVersion)
	cmdutil.Must(err)

	err = ioutil.WriteFile("./buildutil", []byte(contents), 0755)
	cmdutil.Must(err)
}
