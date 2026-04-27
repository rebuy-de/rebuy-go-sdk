package riverutil

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type DatabaseCollector struct {
	pool *pgxpool.Pool

	failingJobs *prometheus.Desc
}

func NewDatabaseCollector(pool *pgxpool.Pool) *DatabaseCollector {
	labels := prometheus.Labels{}

	return &DatabaseCollector{
		pool: pool,
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
	query := `SELECT count(*) FROM river_job WHERE state = 'retryable' AND attempt > 5;`
	err := c.pool.QueryRow(context.Background(), query).Scan(&retryableJobs)
	if err != nil {
		slog.Error("failed to query river retryable jobs", "query", query, "error", err)
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
