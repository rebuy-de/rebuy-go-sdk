package cmdutil

import (
	"context"
	"log/slog"
	"os"

	"github.com/rs/zerolog"
	slogmulti "github.com/samber/slog-multi"
	slogzerolog "github.com/samber/slog-zerolog/v2"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
)

func WithLoggingOptions() Option {
	return WithLoggingOptionsHostname("")
}

func WithLoggingOptionsHostname(hostname string) Option {
	var (
		verboseLogging        bool
		jsonLogging           bool
		openTelemetryProtocol string
		openTelemetryEndpoint string
	)

	return func(cmd *cobra.Command) error {
		cmd.PersistentFlags().BoolVarP(
			&verboseLogging, "verbose", "v", false,
			"Prints debug log messages")

		cmd.PersistentFlags().BoolVar(
			&jsonLogging, "json-logs", false,
			"Print logs in JSON format",
		)

		cmd.PersistentFlags().StringVar(
			&openTelemetryProtocol, "otel-protocol", "",
			`Protocol to use for OpenTelemetry logs (grpc or http)`,
		)

		cmd.PersistentFlags().StringVar(
			&openTelemetryEndpoint, "otel-endpoint", "",
			`Endpoint to use for OpenTelemetry logs (e.g. localhost:4317)`,
		)

		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			var zerologLogger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
			if jsonLogging {
				zerologLogger = zerolog.New(os.Stderr)
			}

			logLevel := slog.LevelInfo
			if verboseLogging {
				logLevel = slog.LevelDebug
			}

			logger := slog.New(slogzerolog.Option{Level: logLevel, Logger: &zerologLogger}.NewZerologHandler())
			slog.SetDefault(logger)

			if openTelemetryEndpoint != "" && openTelemetryProtocol != "" {
				var processor *log.BatchProcessor
				switch openTelemetryProtocol {
				case "grpc":
					exporter, err := otlploggrpc.New(context.Background(), otlploggrpc.WithCompressor("gzip"), otlploggrpc.WithInsecure(), otlploggrpc.WithEndpoint(openTelemetryEndpoint))
					if err != nil {
						slog.Error("Configuring OTEL exporter failed, only using console logger", "protocol", openTelemetryProtocol, "endpoint", openTelemetryEndpoint)
						return
					}
					processor = log.NewBatchProcessor(exporter)
				case "http":
					exporter, err := otlploghttp.New(context.Background(), otlploghttp.WithCompression(otlploghttp.GzipCompression), otlploghttp.WithInsecure(), otlploghttp.WithEndpoint(openTelemetryEndpoint))
					if err != nil {
						slog.Error("Configuring OTEL exporter failed, only using console logger", "protocol", openTelemetryProtocol, "endpoint", openTelemetryEndpoint)
						return
					}
					processor = log.NewBatchProcessor(exporter)
				default:
					slog.Error("Unsupported protocol selected for OTEL, only using console logger", "protocol", openTelemetryProtocol, "endpoint", openTelemetryEndpoint)
					return
				}

				provider := log.NewLoggerProvider(
					log.WithProcessor(processor),
				)

				attributes := []attribute.KeyValue{
					attribute.String("facility", Name),
					attribute.String("version", Version),
					attribute.String("commit-sha", CommitHash),
				}

				if hostname != "" {
					attributes = append(attributes, attribute.String("host", hostname))
				}

				logger = slog.New(
					slogmulti.Fanout(
						otelslog.NewHandler(Name, otelslog.WithLoggerProvider(provider), otelslog.WithAttributes(attributes...)),
						slogzerolog.Option{Level: logLevel, Logger: &zerologLogger}.NewZerologHandler(),
					),
				)
				slog.SetDefault(logger)
			}
		}

		return nil
	}
}
