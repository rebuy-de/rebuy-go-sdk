package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/executil"
)

func call(ctx context.Context, command string, args ...string) error {
	logrus.Debugf("$ %s %s", command, strings.Join(args, " "))
	c := exec.Command(command, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return executil.Run(ctx, c)
}

type BuildParameters struct {
	TargetSystems  []string
	TargetPackages []string
	S3URL          string

	CreateCompressed bool
	CreateRPM        bool
	CreateDEB        bool

	GoCommand string

	CGO bool
	PGO string
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
	cmd.PersistentFlags().StringVar(
		&r.Parameters.PGO, "pgo", "",
		"Sets input for PGO option.")
	cmd.PersistentFlags().StringVar(
		&r.Parameters.GoCommand, "go-command", "go",
		"Which Go command to use.")

	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
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

func (r *Runner) Run(ctx context.Context, _ []string) error {
	defer r.Inst.Durations.Steps.Stopwatch("all")()

	return runSeq(ctx,
		r.collectInfo,
		r.runVendor,
		r.runTest,
		r.runBuild,
	)
}

func (r *Runner) collectInfo(ctx context.Context) error {
	defer r.Inst.Durations.Steps.Stopwatch("info")()

	info, err := CollectBuildInformation(context.Background(), r.Parameters)
	if err != nil {
		return err
	}

	dumpJSON(info)
	if len(info.Commit.DirtyFiles) > 0 {
		logrus.Warn("The repository contains uncommitted files!")
	}

	r.Info = info
	return nil
}

func (r *Runner) runVendor(ctx context.Context) error {
	defer r.Inst.Durations.Steps.Stopwatch("vendor")()

	if r.Info.Go.Work == "" {
		return call(ctx, r.Parameters.GoCommand, "mod", "vendor")
	}

	return call(ctx, r.Parameters.GoCommand, "work", "vendor")
}

func runSeq(ctx context.Context, fns ...func(ctx context.Context) error) error {
	for _, fn := range fns {
		err := fn(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) runTest(ctx context.Context) error {
	defer r.Inst.Durations.Steps.Stopwatch("test")()

	return runSeq(ctx,
		r.runTestFormat,
		r.runTestStaticcheck,
		r.runTestVet,
		r.runTestPackages,
	)
}

func (r *Runner) runTestFormat(ctx context.Context) error {
	a := []string{"-s", "-l"}
	a = append(a, r.Info.Test.Files...)

	logrus.Info("Testing file formatting (gofmt)")
	defer r.Inst.Durations.Testing.Stopwatch("fmt")()
	return call(ctx, "gofmt", a...)
}

func (r *Runner) runTestStaticcheck(ctx context.Context) error {
	fail := []string{
		"all",
		"-SA1019", // Using a deprecated function, variable, constant or field
	}

	logrus.Info("Testing staticcheck")
	defer r.Inst.Durations.Testing.Stopwatch("staticcheck")()
	return call(ctx, r.Parameters.GoCommand,
		"run", "honnef.co/go/tools/cmd/staticcheck",
		"-f", "stylish",
		"-fail", strings.Join(fail, ","),
		"./...",
	)
}

func (r *Runner) runTestVet(ctx context.Context) error {
	a := []string{"vet"}
	a = append(a, r.Info.Test.Packages...)

	logrus.Info("Testing suspicious constructs (go vet)")
	defer r.Inst.Durations.Testing.Stopwatch("vet")()
	return call(ctx, r.Parameters.GoCommand, a...)
}

func (r *Runner) runTestPackages(ctx context.Context) error {
	a := []string{"test"}
	a = append(a, r.Info.Test.Packages...)

	logrus.Info("Testing packages")
	defer r.Inst.Durations.Testing.Stopwatch("packages")()
	return call(ctx, r.Parameters.GoCommand, a...)
}

func (r *Runner) runBuild(ctx context.Context) error {
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
			{name: "SDKVersion", value: r.Info.SDKVersion.String()},
			{name: "BuildDate", value: r.Info.BuildDate},
			{name: "CommitDate", value: r.Info.Commit.Date},
			{name: "CommitHash", value: r.Info.Commit.Hash},
		}

		ldFlags := []string{}
		for _, entry := range ldData {
			ldFlags = append(ldFlags, fmt.Sprintf(
				`-X 'github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil.%s=%s'`,
				entry.name, entry.value,
			))
		}

		cgoEnabled := "0"
		if target.CGO {
			cgoEnabled = "1"
		}

		err := errors.Join(
			os.Setenv("GOOS", target.System.OS),
			os.Setenv("GOARCH", target.System.Arch),
			os.Setenv("CGO_ENABLED", cgoEnabled),
		)
		if err != nil {
			return fmt.Errorf("set env vars: %w", err)
		}

		buildArgs := []string{
			"build",
			"-o", r.dist(target.Outfile),
			"-ldflags", "-s -w " + strings.Join(ldFlags, " "),
		}

		if target.PGO != "" {
			buildArgs = append(buildArgs, "-pgo", target.PGO)
		}

		buildArgs = append(buildArgs, target.Package)

		sw := r.Inst.Durations.Building.Stopwatch(target.Outfile)
		err = call(ctx, r.Parameters.GoCommand, buildArgs...)
		if err != nil {
			return fmt.Errorf("build %s %v: %w", r.Parameters.GoCommand, buildArgs, err)
		}

		r.Inst.ReadSize(target.Outfile)

		sw()
	}

	for _, artifact := range r.Info.Artifacts {
		// create symlinks
		for _, link := range artifact.Aliases {
			fullLink := r.dist(link)
			os.Remove(fullLink)
			err := os.Symlink(artifact.Filename, fullLink)
			if err != nil {
				return fmt.Errorf("create symlink: %w", err)
			}
		}
	}

	return nil
}
