package cmdutil

import (
	"context"

	graylog "github.com/gemnasium/logrus-graylog-hook/v3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Option func(*cobra.Command) error

func New(use, desc string, options ...Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: desc,
	}

	var (
		preRuns           = make([]func(*cobra.Command, []string), 0)
		persistentPreRuns = make([]func(*cobra.Command, []string), 0)
	)

	for _, o := range options {
		err := o(cmd)
		must(err)

		if cmd.PreRun != nil {
			preRuns = append(preRuns, cmd.PreRun)
		}
		cmd.PreRun = nil

		if cmd.PersistentPreRun != nil {
			persistentPreRuns = append(persistentPreRuns, cmd.PersistentPreRun)
		}

		cmd.PersistentPreRun = nil
	}

	if len(persistentPreRuns) > 0 {
		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			for _, run := range persistentPreRuns {
				run(cmd, args)
			}
		}
	}

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		for _, run := range preRuns {
			run(cmd, args)
		}
	}

	return cmd
}

func WithSubCommand(sub *cobra.Command) Option {
	return func(parent *cobra.Command) error {
		parent.AddCommand(sub)
		return nil
	}
}

// Binder defines the interface used by the generic [WithRun] function.
type Runner interface {
	Bind(*cobra.Command) error
	Run(context.Context, []string) error
}

// WithRunner that accepts a generic type which must implement the [Binder]
// interface. The Bind function gets called with [cobra.Command] so it can
// prepare Cobra flags.
func WithRunner(runner Runner) Option {
	return func(cmd *cobra.Command) error {
		runner.Bind(cmd)

		cmd.Run = func(cmd *cobra.Command, args []string) {
			ctx := SignalRootContext()
			err := runner.Run(ctx, args)
			must(err)
		}
		return nil
	}
}

type LoggerOption struct {
	JSONFormatter bool
	GELFLogger    bool
}

func (o *LoggerOption) Bind(cmd *cobra.Command) error {
	var (
		flagJSON        bool
		flagGELFAddress string
	)

	// Bind json-logs flag, if enabled.
	if o.JSONFormatter {
		cmd.PersistentFlags().BoolVar(
			&flagJSON, "json-logs", false, "Print the logs in JSON format")
	}

	// Bind gelf-address flag, if enabled.
	if o.GELFLogger {
		cmd.PersistentFlags().StringVar(
			&flagGELFAddress, "gelf-address", "",
			`Address to Graylog for logging (format: "ip:port")`)
	}

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if flagJSON {
			logrus.SetFormatter(&logrus.JSONFormatter{
				FieldMap: logrus.FieldMap{
					logrus.FieldKeyTime:  "time",
					logrus.FieldKeyLevel: "level",
					logrus.FieldKeyMsg:   "message",
				},
			})
		}

		if flagGELFAddress != "" {
			hook := graylog.NewGraylogHook(flagGELFAddress,
				map[string]any{
					"uuid":     uuid.New(),
					"facility": Name,
				})
			hook.Level = logrus.DebugLevel
			logrus.AddHook(hook)
		}
	}

	return nil
}
