package cmdutil

import (
	graylog "github.com/gemnasium/logrus-graylog-hook/v3"
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

func WithLogToGraylog() Option {
	return WithLogToGraylogHostname("")
}

func WithLogToGraylogHostname(hostname string) Option {
	var (
		gelfAddress string
	)

	return func(cmd *cobra.Command) error {
		cmd.PersistentFlags().StringVar(
			&gelfAddress, "gelf-address", "",
			`Address to Graylog for logging (format: "ip:port").`)

		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			if gelfAddress == "" {
				return
			}

			hook := graylog.NewGraylogHook(gelfAddress, map[string]interface{}{
				"facility":   Name,
				"version":    Version,
				"commit-sha": CommitHash,
			})

			if hostname != "" {
				hook.Host = hostname
			}

			logrus.AddHook(hook)
		}

		return nil
	}
}
