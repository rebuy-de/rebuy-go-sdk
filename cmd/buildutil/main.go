package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
)

func main() {
	defer cmdutil.HandleExit()
	if err := NewRootCommand().Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func NewRootCommand() *cobra.Command {
	return cmdutil.New(
		"buildutil", "Build tool for Go projects as part of the rebuy-go-sdk",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(slog.LevelDebug),

		cmdutil.WithRunner(new(Runner)),
	)
}
