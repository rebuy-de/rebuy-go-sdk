package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
)

func main() {
	defer cmdutil.HandleExit()
	if err := NewRootCommand().Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func NewRootCommand() *cobra.Command {
	cmd := cmdutil.New(
		"packageutil", "Package tool for Go binaries as part of the rebuy-go-sdk",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),

		cmdutil.WithRunner(new(Runner)),
	)

	cmd.Args = cobra.MinimumNArgs(1)
	cmd.Use = "packageutil [flags] binary-file1 [binary-file2 ...]"

	return cmd
}
