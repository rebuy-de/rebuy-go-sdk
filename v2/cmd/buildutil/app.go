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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"

	"github.com/rebuy-de/rebuy-go-sdk/v2/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v2/executil"
)

func call(ctx context.Context, command string, args ...string) {
	logrus.Infof("$ %s", strings.Join(args, " "))
	c := exec.Command(command, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	cmdutil.Must(executil.Run(ctx, c))
}

type SystemInfo struct {
	OS   string
	Arch string
	Ext  string
}

func (i SystemInfo) FileSufix() string {
	return fmt.Sprintf("%s-%s%s", i.OS, i.Arch, i.Ext)
}

type BuildInfo struct {
	Module    string
	BuildDate string

	System SystemInfo

	Commit struct {
		Hash       string
		Date       string
		Version    string
		DirtyFiles []string `json:",omitempty"`
	}
}

type TargetInfo struct {
	Package string
	Name    string
	System  SystemInfo
}

type App struct {
	Info       BuildInfo
	Parameters struct {
		TargetSystems []string
	}
}

func (app *App) BindBuildFlags(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSliceVarP(
		&app.Parameters.TargetSystems, "cross-compile", "x", []string{},
		"Targets for cross compilation (eg linux/amd64). Can be used multiple times.")
	return nil
}

func (app *App) collectBuildInformation(ctx context.Context) {
	os.Setenv("GOPATH", "")

	logrus.Info("Collecting build information")

	e := NewExecutor(ctx)

	app.Info.BuildDate = time.Now().Format(time.RFC3339)
	app.Info.Module = e.GetString("go", "list", "-m")
	app.Info.System.OS = e.GetString("go", "env", "GOOS")
	app.Info.System.Arch = e.GetString("go", "env", "GOARCH")
	app.Info.System.Ext = e.GetString("go", "env", "GOEXE")
	app.Info.Commit.Version = e.GetString("git", "describe", "--always", "--dirty", "--tags")
	app.Info.Commit.Date = time.Unix(e.GetInt64("git", "show", "-s", "--format=%ct"), 0).Format(time.RFC3339)
	app.Info.Commit.Hash = e.GetString("git", "rev-parse", "HEAD")

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
			{name: "Version", value: app.Info.Commit.Version},
			{name: "GoModule", value: app.Info.Module},
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

		outfile := fmt.Sprintf("%s-%s-%s",
			target.Name, app.Info.Commit.Version,
			target.System.FileSufix())
		link := path.Join("dist", target.Name)
		linkCC := path.Join("dist", fmt.Sprintf("%s-%s",
			target.Name, target.System.FileSufix()))

		os.Remove(linkCC)

		os.Setenv("GOOS", target.System.OS)
		os.Setenv("GOARCH", target.System.Arch)
		call(ctx, "go", "build",
			"-o", path.Join("dist", outfile),
			"-ldflags", strings.Join(ldFlags, " "),
			target.Package)

		cmdutil.Must(os.Symlink(outfile, linkCC))
		if target.System.OS == app.Info.System.OS && target.System.Arch == app.Info.System.Arch {
			os.Remove(link)
			cmdutil.Must(os.Symlink(outfile, link))
		}
	}
}
