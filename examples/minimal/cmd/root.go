package cmd

import (
	"context"
	"fmt"

	"github.com/rebuy-de/rebuy-go-sdk/v6/pkg/cmdutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	return cmdutil.New(
		"minimal", "rebuy-go-sdk-minimal-example",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithLogToGraylog(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),
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

func (r *Runner) Run(ctx context.Context) error {
	fmt.Printf("Hello %s!\n", r.name)
	return nil
}
