package main

import (
	"log/slog"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
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
	return cmdutil.New(
		"buildutil", "Build tool for Go projects as part of the rebuy-go-sdk",
		cmdutil.WithLoggingOptions(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(slog.LevelDebug),

		cmdutil.WithRunner(new(Runner)),
	)
}
