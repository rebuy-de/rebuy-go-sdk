package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

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

type BuildInfo struct {
	Module    string
	BuildDate string

	Go struct {
		OS   string
		Arch string
		Ext  string
	}

	Commit struct {
		Hash    string
		Date    string
		Version string
	}
}

type TargetInfo struct {
	Target  string
	Package string
	Name    string
}

type App struct {
	Info BuildInfo
}

func (app *App) collectBuildInformation(ctx context.Context) {
	os.Setenv("GOPATH", "")

	logrus.Info("Collecting build information")

	e := NewExecutor(ctx)

	app.Info.BuildDate = time.Now().Format(time.RFC3339)
	app.Info.Module = e.GetString("go", "list", "-m")
	app.Info.Go.OS = e.GetString("go", "env", "GOOS")
	app.Info.Go.Arch = e.GetString("go", "env", "GOARCH")
	app.Info.Go.Ext = e.GetString("go", "env", "GOEXE")
	app.Info.Commit.Version = e.GetString("git", "describe", "--always", "--dirty", "--tags")
	app.Info.Commit.Date = time.Unix(e.GetInt64("git", "show", "-s", "--format=%ct"), 0).Format(time.RFC3339)
	app.Info.Commit.Hash = e.GetString("git", "rev-parse", "HEAD")

	cmdutil.Must(e.Err())

	b, err := json.MarshalIndent(app.Info, "", "    ")
	cmdutil.Must(err)
	fmt.Println(string(b))
}

func (app *App) collectTargetInformation(ctx context.Context, target string) TargetInfo {
	logrus.WithField("target", target).Info("Collecting target information")

	info := TargetInfo{}
	info.Target = target
	info.Package = path.Join(app.Info.Module, target)
	info.Name = path.Base(info.Package)

	b, err := json.MarshalIndent(info, "", "    ")
	cmdutil.Must(err)
	fmt.Println(string(b))

	return info
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
	targets := args
	if len(targets) == 0 {
		targets = append(targets, ".")
	}

	app.collectBuildInformation(ctx)

	for _, target := range targets {
		info := app.collectTargetInformation(ctx, target)

		ldData := []struct {
			name  string
			value string
		}{
			{name: "Name", value: info.Name},
			{name: "Version", value: app.Info.Commit.Version},
			{name: "GoModule", value: app.Info.Module},
			{name: "GoPackage", value: info.Package},
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

		outfile := fmt.Sprintf("%s-%s-%s-%s%s",
			info.Name, app.Info.Commit.Version,
			app.Info.Go.OS, app.Info.Go.Arch, app.Info.Go.Ext)
		link := path.Join("dist", info.Name)

		os.Remove(link)
		call(ctx, "go", "build",
			"-o", path.Join("dist", outfile),
			"-ldflags", strings.Join(ldFlags, " "),
			target)

		cmdutil.Must(os.Symlink(outfile, link))
	}
}
