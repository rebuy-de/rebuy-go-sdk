package cmdutil

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func WithLogVerboseFlag() Option {
	var (
		enabled bool
	)

	return func(cmd *cobra.Command) error {
		cmd.PersistentFlags().BoolVarP(
			&enabled, "verbose", "v", false,
			"prints debug log messages")

		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			logrus.SetLevel(logrus.InfoLevel)
			if enabled {
				logrus.SetLevel(logrus.DebugLevel)
			}
		}

		return nil
	}
}
