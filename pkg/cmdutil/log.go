package cmdutil

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Graylog2/go-gelf/gelf"
	"github.com/lmittmann/tint"
	sloggraylog "github.com/samber/slog-graylog/v2"
	slogmulti "github.com/samber/slog-multi"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// logLevel is the shared slog.LevelVar used to dynamically control the log level.
var logLevel = new(slog.LevelVar)

// newCLIHandler creates the appropriate CLI handler based on whether stderr is a TTY.
// On a TTY it uses tint for colorized output, otherwise it falls back to slog.JSONHandler.
func newCLIHandler() slog.Handler {
	if term.IsTerminal(int(os.Stderr.Fd())) {
		addSource := false
		fmt.Printf("%v", logLevel.Level())
		if logLevel.Level() == slog.LevelDebug {
			addSource = true
		}
		return tint.NewHandler(os.Stderr, &tint.Options{
			Level:      logLevel,
			TimeFormat: time.TimeOnly,
			AddSource:  addSource,
		})
	}
	return tint.NewHandler(os.Stderr, &tint.Options{
		Level:      logLevel,
		TimeFormat: time.DateTime,
		NoColor:    true,
	})
}

func init() {
	logLevel.Set(slog.LevelInfo)
	slog.SetDefault(slog.New(newCLIHandler()))
}

func WithLogVerboseFlag() Option {
	var (
		enabled bool
	)

	return func(cmd *cobra.Command) error {
		cmd.PersistentFlags().BoolVarP(
			&enabled, "verbose", "v", false,
			"prints debug log messages")

		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			logLevel.Set(slog.LevelInfo)
			if enabled {
				fmt.Printf("%v", logLevel.Level())
				logLevel.Set(slog.LevelDebug)
				fmt.Printf("%v", logLevel.Level())
			}
		}
		fmt.Printf("%v", logLevel.Level())
		slog.SetDefault(slog.New(newCLIHandler()))

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

			gelfWriter, err := gelf.NewWriter(gelfAddress)
			if err != nil {
				slog.Error("failed to create GELF writer", "error", err, "address", gelfAddress)
				return
			}

			graylogHandler := sloggraylog.Option{
				Level:  logLevel,
				Writer: gelfWriter,
			}.NewGraylogHandler()

			handler := slogmulti.Fanout(
				newCLIHandler(),
				graylogHandler,
			)

			logger := slog.New(handler).With(
				"facility", Name,
				"version", Version,
				"commit-sha", CommitHash,
			)

			slog.SetDefault(logger)
		}

		return nil
	}
}
