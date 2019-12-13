package main

import (
	"context"
	"io/ioutil"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type Runner struct {
	TargetVersion string
}

func (r *Runner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&r.TargetVersion, "target-version", "",
		"Target version for the rebuy-go-sdk")
	return nil
}

func (r *Runner) RunGenerateWrapper(ctx context.Context, cmd *cobra.Command, args []string) {
	contents, err := generateWrapper(r.TargetVersion)
	cmdutil.Must(err)

	err = ioutil.WriteFile("./buildutil", []byte(contents), 0755)
	cmdutil.Must(err)
}
