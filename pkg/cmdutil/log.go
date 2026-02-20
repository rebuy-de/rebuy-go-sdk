package cmdutil

import (
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

// logJSON controls whether JSON output is forced.
var logJSON bool

// logAddSource controls whether source location is included in log output.
var logAddSource bool

// newCLIHandler creates the appropriate CLI handler based on configuration.
// If logJSON is true, it always uses slog.JSONHandler.
// Otherwise, on a TTY it uses tint for colorized output; on a non-TTY it uses
// tint without color and with a longer timestamp.
func newCLIHandler() slog.Handler {
	if logJSON {
		return slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level:     logLevel,
			AddSource: logAddSource,
		})
	}

	if term.IsTerminal(int(os.Stderr.Fd())) {
		return tint.NewHandler(os.Stderr, &tint.Options{
			Level:      logLevel,
			TimeFormat: time.TimeOnly,
			AddSource:  logAddSource,
		})
	}

	return tint.NewHandler(os.Stderr, &tint.Options{
		Level:      logLevel,
		TimeFormat: time.DateTime,
		NoColor:    true,
		AddSource:  logAddSource,
	})
}

// reconfigureLogger rebuilds the default logger with the current settings.
// This must be called after any change to logLevel, logJSON, or logAddSource.
func reconfigureLogger() {
	slog.SetDefault(slog.New(newCLIHandler()))
}

func init() {
	logLevel.Set(slog.LevelInfo)
	reconfigureLogger()
}

// WithLogVerboseFlag adds a -v/--verbose flag that sets the log level to debug
// and enables source location in log output.
func WithLogVerboseFlag() Option {
	var enabled bool

	return func(cmd *cobra.Command) error {
		cmd.PersistentFlags().BoolVarP(
			&enabled, "verbose", "v", false,
			"prints debug log messages")

		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			if enabled {
				logLevel.Set(slog.LevelDebug)
				logAddSource = true
			} else {
				logLevel.Set(slog.LevelInfo)
				logAddSource = false
			}
			reconfigureLogger()
		}

		return nil
	}
}

// WithLogJSONFlag adds a --log-json flag that forces JSON log output
// regardless of whether the output is a TTY.
func WithLogJSONFlag() Option {
	return func(cmd *cobra.Command) error {
		cmd.PersistentFlags().BoolVar(
			&logJSON, "log-json", false,
			"force JSON log output")

		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			reconfigureLogger()
		}

		return nil
	}
}

func WithLogToGraylog() Option {
	return WithLogToGraylogHostname("")
}

func WithLogToGraylogHostname(hostname string) Option {
	var gelfAddress string

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
