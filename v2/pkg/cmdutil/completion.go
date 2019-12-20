package cmdutil

import (
	"os"

	"github.com/spf13/cobra"
)

func WithCompletionCommand() Option {
	return func(cmd *cobra.Command) error {
		cmd.GenBashCompletion(os.Stdout)
		return nil
	}
}
