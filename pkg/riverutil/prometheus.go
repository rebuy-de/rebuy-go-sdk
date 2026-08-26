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
			`SELECT count(*) FROM %s.river_job WHERE state = 'retryable' AND attempt > 5;`,
			pgx.Identifier{string(schema)}.Sanitize(),
		),
		failingJobs: prometheus.NewDesc(
			"rebuy_go_sdk_river_failing_jobs",
			"Number of River jobs that were retried more than 5 times",
			nil,
			labels,
		),
	}
}

func (c *DatabaseCollector) Collect(ch chan<- prometheus.Metric) {
	var retryableJobs int64
	err := c.pool.QueryRow(context.Background(), c.failingJobsQuery).Scan(&retryableJobs)
	if err != nil {
		slog.Error("failed to query river retryable jobs", "query", c.failingJobsQuery, "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.failingJobs,
		prometheus.GaugeValue,
		float64(retryableJobs),
	)
}

func (c *DatabaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.failingJobs
}
