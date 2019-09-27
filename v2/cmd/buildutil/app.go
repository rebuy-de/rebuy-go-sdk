package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"

	"github.com/rebuy-de/rebuy-go-sdk/v2/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v2/executil"
)

func call(ctx context.Context, command string, args ...string) {
	logrus.Infof("$ %s %s", command, strings.Join(args, " "))
	c := exec.Command(command, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	cmdutil.Must(executil.Run(ctx, c))
}

type Version struct {
	Major, Minor, Patch int
	PreRelease          string
}

func ParseVersion(s string) (Version, error) {
	var (
		v   Version
		err error
	)

	s = strings.ReplaceAll(s, "-", ".")
	p := strings.Split(s, ".")

	if len(p) < 3 {
		return Version{}, errors.Errorf("invalid version '%s': not enough parts", s)
	}

	v.Major, err = strconv.Atoi(strings.TrimLeft(p[0], "v"))
	if err != nil {
		return Version{}, errors.WithStack(err)
	}

	v.Minor, err = strconv.Atoi(p[1])
	if err != nil {
		return Version{}, errors.WithStack(err)
	}

	v.Patch, err = strconv.Atoi(p[2])
	if err != nil {
		return Version{}, errors.WithStack(err)
	}

	if len(p) > 3 {
		v.PreRelease = strings.Join(p[3:], "-")
	}

	return v, nil
}

func (v Version) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s = fmt.Sprintf("%s-%s", s, v.PreRelease)
	}
	return s
}

type SystemInfo struct {
	OS   string
	Arch string
	Ext  string `json:",omitempty"`
}

func (i SystemInfo) FileSufix() string {
	return fmt.Sprintf("%s-%s%s", i.OS, i.Arch, i.Ext)
}

type BuildInfo struct {
	BuildDate string
	System    SystemInfo
	Version   Version

	Go struct {
		Module string
		Dir    string
	}

	Commit struct {
		Hash       string
		Branch     string
		Date       string
		DirtyFiles []string `json:",omitempty"`
	}
}

type TargetInfo struct {
	Package string
	Name    string
	System  SystemInfo

	Outfile struct {
		Name    string
		Aliases []string
	}
}

type App struct {
	Info       BuildInfo
	Parameters struct {
		TargetSystems []string
		S3Bucket      string
	}
}

func (app *App) BindBuildFlags(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSliceVarP(
		&app.Parameters.TargetSystems, "cross-compile", "x", []string{},
		"Targets for cross compilation (eg linux/amd64). Can be used multiple times.")
	cmd.PersistentFlags().StringVar(
		&app.Parameters.S3Bucket, "s3-bucket", "",
		"S3 Bucket to upload compiled releases.")
	return nil
}

func (app *App) collectBuildInformation(ctx context.Context) {
	var err error

	os.Setenv("GOPATH", "")

	logrus.Info("Collecting build information")

	e := NewExecutor(ctx)

	app.Info.BuildDate = time.Now().Format(time.RFC3339)
	app.Info.Go.Module = e.GetString("go", "list", "-m")
	app.Info.Go.Dir = e.GetString("go", "list", "-m", "-f", "{{.Dir}}")
	app.Info.System.OS = e.GetString("go", "env", "GOOS")
	app.Info.System.Arch = e.GetString("go", "env", "GOARCH")
	app.Info.System.Ext = e.GetString("go", "env", "GOEXE")
	app.Info.Commit.Date = time.Unix(e.GetInt64("git", "show", "-s", "--format=%ct"), 0).Format(time.RFC3339)
	app.Info.Commit.Hash = e.GetString("git", "rev-parse", "HEAD")
	app.Info.Commit.Branch = e.GetString("git", "rev-parse", "--abbrev-ref", "HEAD")

	app.Info.Version, err = ParseVersion(e.GetString("git", "describe", "--always", "--dirty", "--tags"))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse version")
	}

	status := strings.TrimSpace(e.GetString("git", "status", "-s"))
	if status != "" {
		app.Info.Commit.DirtyFiles = strings.Split(status, "\n")
	}

	cmdutil.Must(e.Err())

	dumpJSON(app.Info)

	if len(app.Info.Commit.DirtyFiles) > 0 {
		logrus.Warn("The repository contains uncommitted files!")
	}
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

func (app *App) RunBuild(ctx context.Context, cmd *cobra.Command, args []string) {
	app.collectBuildInformation(ctx)

	if len(args) == 0 {
		logrus.Info("No targets specified. Discovering all packages.")
		args = []string{"./..."}
	}

	targetSystems := []SystemInfo{}
	for _, target := range app.Parameters.TargetSystems {
		parts := strings.Split(target, "/")
		if len(parts) != 2 {
			logrus.Errorf("Invalid format for cross compiling target '%s'.", target)
			cmdutil.Exit(1)
		}

		info := SystemInfo{}
		info.OS = parts[0]
		info.Arch = parts[1]
		if info.OS == "windows" {
			info.Ext = ".exe"
		}

		targetSystems = append(targetSystems, info)
	}

	if len(targetSystems) == 0 {
		logrus.Info("No cross compiling targets specified. Using local machine.")
		targetSystems = append(targetSystems, app.Info.System)
	}

	targets := []TargetInfo{}
	for _, arg := range args {
		pkgs, err := packages.Load(nil, arg)
		cmdutil.Must(err)

		for _, pkg := range pkgs {
			if pkg.Name != "main" {
				continue
			}
			logrus.Infof("Found Package %s", pkg.PkgPath)

			for _, targetSystem := range targetSystems {
				info := TargetInfo{
					Package: pkg.PkgPath,
					Name:    path.Base(pkg.PkgPath),
					System:  targetSystem,
				}

				info.Outfile.Name = fmt.Sprintf("%s-%s-%s",
					info.Name, app.Info.Version.String(),
					info.System.FileSufix())

				info.Outfile.Aliases = []string{
					fmt.Sprintf("%s-%s", info.Name, info.System.FileSufix()),
				}

				if info.System.OS == app.Info.System.OS && info.System.Arch == app.Info.System.Arch {
					info.Outfile.Aliases = append(info.Outfile.Aliases, info.Name)
				}

				targets = append(targets, info)
			}
		}
	}

	for _, target := range targets {
		logrus.Infof("Building %s", target.Package)
		dumpJSON(target)

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

		dist := path.Join(app.Info.Go.Dir, "dist")

		call(ctx, "go", "build",
			"-o", path.Join(dist, target.Outfile.Name),
			"-ldflags", strings.Join(ldFlags, " "),
			target.Package)

		for _, link := range target.Outfile.Aliases {
			fullLink := path.Join(dist, link)
			os.Remove(fullLink)
			cmdutil.Must(os.Symlink(target.Outfile.Name, fullLink))
		}
	}

	if app.Parameters.S3Bucket != "" {
		sess, err := session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		})
		cmdutil.Must(err)

		uploader := s3manager.NewUploader(sess)
		dist := path.Join(app.Info.Go.Dir, "dist")

		for _, target := range targets {
			logrus.Infof("Uploading %s to s3://%s/", target.Outfile.Name, app.Parameters.S3Bucket)

			f, err := os.Open(path.Join(dist, target.Outfile.Name))
			cmdutil.Must(err)

			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket: &app.Parameters.S3Bucket,
				Key:    &target.Outfile.Name,
				Body:   f,

				Metadata: map[string]*string{
					"GoModule":  aws.String(app.Info.Go.Module),
					"GoPackage": aws.String(target.Package),
					"Branch":    aws.String(app.Info.Commit.Branch),
					"System":    aws.String(fmt.Sprintf("%s/%s", target.System.OS, target.System.Arch)),
				},
			})
			cmdutil.Must(err)
		}
	}
}
