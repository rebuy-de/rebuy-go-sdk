package main

import (
	"github.com/rebuy-de/rebuy-go-sdk/v2/cmdutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	defer cmdutil.HandleExit()
	if err := NewRootCommand().Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func NewRootCommand() *cobra.Command {
	app := new(App)

	return cmdutil.New(
		"rebuy-buildutil", "Build tool for Go projects as part of the rebuy-go-sdk",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),

		app.Bind,
		cmdutil.WithRun(app.RunAll),

		cmdutil.WithSubCommand(cmdutil.New(
			"vendor", "Update vendor directory",
			cmdutil.WithRun(app.RunVendor),
		)),
		cmdutil.WithSubCommand(cmdutil.New(
			"build", "Build binaries",
			cmdutil.WithRun(app.RunBuild),
		)),
		cmdutil.WithSubCommand(cmdutil.New(
			"upload", "Upload artifacts to S3",
			cmdutil.WithRun(app.RunUpload),
		)),
		cmdutil.WithSubCommand(cmdutil.New(
			"clean", "Clean workspace",
			cmdutil.WithRun(app.RunClean),
		)),
	)
}
