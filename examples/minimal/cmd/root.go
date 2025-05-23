package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	return cmdutil.New(
		"minimal", "rebuy-go-sdk-minimal-example",
		cmdutil.WithLoggingOptions(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(slog.LevelDebug),
		cmdutil.WithRunner(new(Runner)),
	)
}

type Runner struct {
	name string
}

func (r *Runner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&r.name, "name", "World",
		`Your name.`)
	return nil
}

func (r *Runner) Run(ctx context.Context, _ []string) error {
	fmt.Printf("Hello %s!\n", r.name)
	return nil
}
