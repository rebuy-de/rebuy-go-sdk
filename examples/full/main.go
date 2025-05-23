package main

import (
	"log/slog"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/cmd"
)

func main() {
	defer cmdutil.HandleExit()
	if err := cmd.NewRootCommand().Execute(); err != nil {
		slog.Error("Command execution failed", "error", err)
		cmdutil.Exit(cmdutil.ExitCodeGeneralError)
	}
}