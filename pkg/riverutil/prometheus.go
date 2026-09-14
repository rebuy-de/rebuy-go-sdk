package riverutil

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/pgutil"
)

type DatabaseCollector struct {
	pool             *pgxpool.Pool
	failingJobsQuery string

	failingJobs *prometheus.Desc
}

func NewDatabaseCollector(pool *pgxpool.Pool, schema pgutil.Schema) *DatabaseCollector {
	labels := prometheus.Labels{}

	return &DatabaseCollector{
		pool: pool,
		failingJobsQuery: fmt.Sprintf(
			`SELECT kind, count(*) FROM %s.river_job WHERE state = 'retryable' AND attempt > 5 GROUP BY kind;`,
			pgx.Identifier{string(schema)}.Sanitize(),
		),
		failingJobs: prometheus.NewDesc(
			"rebuy_go_sdk_river_failing_jobs",
			"Number of River jobs that were retried more than 5 times",
			[]string{"kind"},
			labels,
		),
	}
}

func (c *DatabaseCollector) Collect(ch chan<- prometheus.Metric) {
	rows, err := c.pool.Query(context.Background(), c.failingJobsQuery)
	if err != nil {
		slog.Error("failed to query river retryable jobs", "query", c.failingJobsQuery, "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var kind string
		var retryableJobs int64

		err := rows.Scan(&kind, &retryableJobs)
		if err != nil {
			slog.Error("failed to scan river retryable jobs row", "query", c.failingJobsQuery, "error", err)
			return
		}

		ch <- prometheus.MustNewConstMetric(
			c.failingJobs,
			prometheus.GaugeValue,
			float64(retryableJobs),
			kind,
		)
	}

	err = rows.Err()
	if err != nil {
		slog.Error("failed to iterate river retryable jobs rows", "query", c.failingJobsQuery, "error", err)
	}
}

func (c *DatabaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.failingJobs
}
