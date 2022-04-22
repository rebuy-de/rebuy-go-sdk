package cmd

import (
	"context"
	"fmt"

	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/cmdutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	runner := new(Runner)

	return cmdutil.New(
		"minimal", "rebuy-go-sdk-minimal-example",
		runner.Bind,
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithLogToGraylog(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),
		cmdutil.WithRun(runner.Run),
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

func (r *Runner) Run(ctx context.Context, cmd *cobra.Command, args []string) {
	fmt.Printf("Hello %s!\n", r.name)
}
