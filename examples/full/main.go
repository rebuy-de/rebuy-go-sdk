package main

import (
	"log/slog"
	"os"

	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/cmd"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
)

func main() {
	defer cmdutil.HandleExit()
	if err := cmd.NewRootCommand().Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
