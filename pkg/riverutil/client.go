package riverutil

import (
	ddotel "github.com/DataDog/dd-trace-go/v2/ddtrace/opentelemetry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
)

func NewRiverClient(
	pool *pgxpool.Pool,
	tracerProvider *ddotel.TracerProvider,
	configer ConfigGroup,
) (*river.Client[pgx.Tx], error) {
	driver := riverpgxv5.New(pool)
	middlewares := []rivertype.Middleware{}
	if tracerProvider != nil {
		middlewares = append(middlewares, otelriver.NewMiddleware(&otelriver.MiddlewareConfig{
			TracerProvider:              tracerProvider,
			EnableWorkSpanJobKindSuffix: true,
		}))
	}

	prometheus.MustRegister(NewDatabaseCollector(pool))

	middlewares = append(middlewares, new(logutilMiddleware))

	config := &river.Config{
		Workers:    river.NewWorkers(),
		Middleware: middlewares,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 4},
		},
	}

	for _, c := range configer.All {
		err := c.Config(config)
		if err != nil {
			return nil, err
		}
	}

	return river.NewClient(driver, config)
}
