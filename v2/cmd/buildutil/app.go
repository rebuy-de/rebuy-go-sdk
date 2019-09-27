package main

import (
	"context"
	"fmt"
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

	"github.com/rebuy-de/rebuy-go-sdk/v2/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v2/executil"
)

func call(ctx context.Context, command string, args ...string) {
	logrus.Debugf("$ %s %s", command, strings.Join(args, " "))
	c := exec.Command(command, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	cmdutil.Must(executil.Run(ctx, c))
}

type App struct {
	Info       BuildInfo
	Parameters struct {
		TargetSystems  []string
		TargetPackages []string
		S3Bucket       string
	}
}

func (app *App) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSliceVarP(
		&app.Parameters.TargetSystems, "cross-compile", "x", []string{},
		"Targets for cross compilation (eg linux/amd64). Can be used multiple times.")
	cmd.PersistentFlags().StringSliceVarP(
		&app.Parameters.TargetSystems, "package", "p", []string{},
		"Packages to build.")
	cmd.PersistentFlags().StringVar(
		&app.Parameters.S3Bucket, "s3-bucket", "",
		"S3 Bucket to upload compiled releases.")

	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		app.collectBuildInformation(context.Background())
	}

	return nil
}

func (app *App) collectBuildInformation(ctx context.Context) {
	info, err := CollectBuildInformation(ctx,
		app.Parameters.TargetPackages,
		app.Parameters.TargetSystems,
	)
	cmdutil.Must(err)

	dumpJSON(info)
	if len(info.Commit.DirtyFiles) > 0 {
		logrus.Warn("The repository contains uncommitted files!")
	}

	app.Info = info
}

func (app *App) RunAll(ctx context.Context, cmd *cobra.Command, args []string) {
	app.RunVendor(ctx, cmd, args)
	app.RunTest(ctx, cmd, args)
	app.RunBuild(ctx, cmd, args)
	app.RunUpload(ctx, cmd, args)
}

func (app *App) RunClean(ctx context.Context, cmd *cobra.Command, args []string) {
	files, err := filepath.Glob("dist/*")
	cmdutil.Must(err)

	for _, file := range files {
		logrus.Info("remove ", file)
		os.Remove(file)
	}
}

func (app *App) RunVendor(ctx context.Context, cmd *cobra.Command, args []string) {
	call(ctx, "go", "mod", "vendor")
}

func (app *App) RunTest(ctx context.Context, cmd *cobra.Command, args []string) {
	app.RunTestFormat(ctx, cmd, args)
	app.RunTestVet(ctx, cmd, args)
	app.RunTestPackages(ctx, cmd, args)
}

func (app *App) RunTestFormat(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"-s", "-l"}
	a = append(a, app.Info.Test.Files...)

	logrus.Info("Testing file formatting (gofmt)")
	start := time.Now()
	call(ctx, "gofmt", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (app *App) RunTestVet(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"vet"}
	a = append(a, app.Info.Test.Packages...)

	logrus.Info("Testing suspicious constructs (go vet)")
	start := time.Now()
	call(ctx, "go", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (app *App) RunTestPackages(ctx context.Context, cmd *cobra.Command, args []string) {
	a := []string{"test"}
	a = append(a, app.Info.Test.Packages...)

	logrus.Info("Testing packages")
	start := time.Now()
	call(ctx, "go", a...)
	logrus.Infof("Test finished in %v", time.Since(start).Truncate(10*time.Millisecond))
}

func (app *App) RunBuild(ctx context.Context, cmd *cobra.Command, args []string) {
	for _, target := range app.Info.Targets {
		logrus.Infof("Building %s for %s", target.Package, target.System.Name())

		ldData := []struct {
			name  string
			value string
		}{
			{name: "Name", value: target.Name},
			{name: "Version", value: app.Info.Version.String()},
			{name: "GoModule", value: app.Info.Go.Module},
			{name: "GoPackage", value: target.Package},
			{name: "BuildDate", value: app.Info.BuildDate},
			{name: "CommitDate", value: app.Info.Commit.Date},
			{name: "CommitHash", value: app.Info.Commit.Hash},
		}

		ldFlags := []string{}
		for _, entry := range ldData {
			ldFlags = append(ldFlags, fmt.Sprintf(
				`-X 'github.com/rebuy-de/rebuy-go-sdk/v2/cmdutil.%s=%s'`,
				entry.name, entry.value,
			))
		}

		os.Setenv("GOOS", target.System.OS)
		os.Setenv("GOARCH", target.System.Arch)
		os.Setenv("CGO_ENABLED", "0")

		dist := path.Join(app.Info.Go.Dir, "dist")

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

func (app *App) RunUpload(ctx context.Context, cmd *cobra.Command, args []string) {
	if app.Parameters.S3Bucket == "" {
		logrus.Warn("No S3 Bucket specified. Skipping upload.")
		return
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	cmdutil.Must(err)

	uploader := s3manager.NewUploader(sess)
	dist := path.Join(app.Info.Go.Dir, "dist")

	for _, target := range app.Info.Targets {
		logrus.Infof("Uploading %s to s3://%s/", target.Outfile.Name, app.Parameters.S3Bucket)

		f, err := os.Open(path.Join(dist, target.Outfile.Name))
		cmdutil.Must(err)

		start := time.Now()
		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket: &app.Parameters.S3Bucket,
			Key:    &target.Outfile.Name,
			Body:   f,

			Metadata: map[string]*string{
				"GoModule":  aws.String(app.Info.Go.Module),
				"GoPackage": aws.String(target.Package),
				"Branch":    aws.String(app.Info.Commit.Branch),
				"System":    aws.String(target.System.Name()),
			},
		})
		cmdutil.Must(err)

		logrus.Infof("Upload finished in %v", time.Since(start).Truncate(10*time.Millisecond))
	}
}
