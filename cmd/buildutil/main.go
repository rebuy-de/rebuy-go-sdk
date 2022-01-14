package main

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/cmdutil"
)

func main() {
	defer cmdutil.HandleExit()
	if err := NewRootCommand().Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func NewRootCommand() *cobra.Command {
	runner := new(Runner)

	return cmdutil.New(
		"buildutil", "Build tool for Go projects as part of the rebuy-go-sdk",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),

		runner.Bind,
		cmdutil.WithRun(runner.RunAll),

		cmdutil.WithSubCommand(cmdutil.New(
			"info", "Show project info",
			// Info output is already done by the prerun, therefore we do not
			// need to actually do anything.
			cmdutil.WithRun(func(ctx context.Context, cmd *cobra.Command, args []string) {}),
		)),

		cmdutil.WithSubCommand(cmdutil.New(
			"vendor", "Update vendor directory",
			cmdutil.WithRun(runner.RunVendor),
		)),
		cmdutil.WithSubCommand(cmdutil.New(
			"test", "Run unit tests",
			cmdutil.WithRun(runner.RunTest),
			cmdutil.WithSubCommand(cmdutil.New(
				"fmt", "Tests file formatting",
				cmdutil.WithRun(runner.RunTestFormat),
			)),
			cmdutil.WithSubCommand(cmdutil.New(
				"vet", "Tests for suspicious constructs",
				cmdutil.WithRun(runner.RunTestVet),
			)),
			cmdutil.WithSubCommand(cmdutil.New(
				"packages", "Tests Packages",
				cmdutil.WithRun(runner.RunTestPackages),
			)),
		)),
		cmdutil.WithSubCommand(cmdutil.New(
			"build", "Build binaries",
			cmdutil.WithRun(runner.RunBuild),
		)),
		cmdutil.WithSubCommand(cmdutil.New(
			"artifacts", "Create artifacts",
			cmdutil.WithRun(runner.RunArtifacts),
		)),
		cmdutil.WithSubCommand(cmdutil.New(
			"upload", "Upload artifacts to S3",
			cmdutil.WithRun(runner.RunUpload),
		)),
		cmdutil.WithSubCommand(cmdutil.New(
			"clean", "Clean workspace",
			cmdutil.WithRun(runner.RunClean),
		)),
	)
}
