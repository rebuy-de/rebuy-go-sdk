package cmdutil

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	graylog "gopkg.in/gemnasium/logrus-graylog-hook.v2"
)

// ApplicationRunner is an optional interface for NewRootCommand.
type ApplicationRunner interface {
	// Run contains the actual application code. It is equivalent to
	// the Run command of Cobra.
	Run(cmd *cobra.Command, args []string)
}

// ApplicationRunnerWithContext is an optional interface for NewRootCommand.
type ApplicationRunnerWithContext interface {
	// Run contains the actual application code. It is equivalent to
	// the Run command of Cobra plus adding a context. The context gets
	// cancelled, if the application receives a SIGINT or SIGTERM.
	Run(ctx context.Context, cmd *cobra.Command, args []string)
}

// ApplicationBinder is an optional interface for NewRootCommand.
type ApplicationBinder interface {

	// Bind is used to bind command line flags to fields of the
	// application struct.
	Bind(cmd *cobra.Command)
}

// NewRootCommand creates a Cobra command, which reflects our current best
// practices. It adds a verbose flag, sets up logrus and registers a Graylog
// hook. Also it registers the NewVersionCommand and prints the version on
// startup. The provided app might implement ApplicationRunner and
// ApplicationBinder.
func NewRootCommand(app interface{}) *cobra.Command {
	var (
		gelfAddress string
		verbose     bool
	)

	var run func(cmd *cobra.Command, args []string)

	// Note: since ApplicationRunnerWithContext and ApplicationRunner require
	// the same function Run with different parameters, they are mutually
	// exclusive.
	runner, ok := app.(ApplicationRunner)
	if ok {
		run = runner.Run
	}

	runnerWithContext, ok := app.(ApplicationRunnerWithContext)
	if ok {
		run = func(cmd *cobra.Command, args []string) {
			runnerWithContext.Run(SignalRootContext(), cmd, args)
		}
	}

	cmd := &cobra.Command{
		Use: BuildName,
		Run: run,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logrus.SetLevel(logrus.InfoLevel)

			if verbose {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if gelfAddress != "" {
				labels := map[string]interface{}{
					"facility":   BuildName,
					"version":    BuildVersion,
					"commit-sha": BuildHash,
				}
				hook := graylog.NewGraylogHook(gelfAddress, labels)
				hook.Level = logrus.DebugLevel
				logrus.AddHook(hook)
			}

			logrus.WithFields(logrus.Fields{
				"Version": BuildVersion,
				"Date":    BuildDate,
				"Commit":  BuildHash,
			}).Infof("%s started", BuildName)
		},

		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			logrus.Infof("%s stopped", BuildName)
		},
	}

	cmd.PersistentFlags().BoolVarP(
		&verbose, "verbose", "v", false,
		`Show debug logs.`)
	cmd.PersistentFlags().StringVar(
		&gelfAddress, "gelf-address", "",
		`Address to Graylog for logging (format: "ip:port").`)

	binder, ok := app.(ApplicationBinder)
	if ok {
		binder.Bind(cmd)
	}

	cmd.AddCommand(NewVersionCommand())

	return cmd
}
