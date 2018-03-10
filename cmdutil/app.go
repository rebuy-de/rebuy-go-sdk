package cmdutil

import (
	"github.com/spf13/cobra"
	graylog "gopkg.in/gemnasium/logrus-graylog-hook.v2"

	log "github.com/sirupsen/logrus"
)

// Application provides the basic behaviour for NewRootCommand.
type Application interface {
	// Run contains the actual application code. It is equivalent to
	// the Run command of Cobra.
	Run(cmd *cobra.Command, args []string)

	// Bind is used to bind command line flags to fields of the
	// application struct.
	Bind(cmd *cobra.Command)
}

// NewRootCommand creates a Cobra command, which reflects our current best
// practices. It adds a verbose flag, sets up logrus and registers a Graylog
// hook. Also it registers the NewVersionCommand and prints the version on
// startup.
func NewRootCommand(app Application) *cobra.Command {
	var (
		gelfAddress string
		verbose     bool
	)

	cmd := &cobra.Command{
		Use: BuildName,
		Run: app.Run,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetLevel(log.InfoLevel)

			if verbose {
				log.SetLevel(log.DebugLevel)
			}

			if gelfAddress != "" {
				labels := map[string]interface{}{
					"facility":   BuildName,
					"version":    BuildVersion,
					"commit-sha": BuildHash,
				}
				hook := graylog.NewGraylogHook(gelfAddress, labels)
				hook.Level = log.DebugLevel
				log.AddHook(hook)
			}

			log.WithFields(log.Fields{
				"Version": BuildVersion,
				"Date":    BuildDate,
				"Commit":  BuildHash,
			}).Infof("%s started", BuildName)
		},

		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			log.Infof("%s stopped", BuildName)
		},
	}

	cmd.PersistentFlags().BoolVarP(
		&verbose, "verbose", "v", false,
		`Show debug logs.`)
	cmd.PersistentFlags().StringVar(
		&gelfAddress, "gelf-address", "",
		`Address to Graylog for logging (format: "ip:port").`)

	app.Bind(cmd)

	cmd.AddCommand(NewVersionCommand())

	return cmd
}
