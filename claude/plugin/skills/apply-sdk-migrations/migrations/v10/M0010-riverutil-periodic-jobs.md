---
id: M0010
title: Adopt riverutil for periodic and background jobs
date: 2026-04-27
sdk_version: v10
type: minor
---

# Adopt riverutil for periodic and background jobs

## Reasoning

The SDK now ships `pkg/riverutil`, which integrates [River](https://riverqueue.com) — a Postgres-backed durable job queue — with the SDK's dependency injection, structured logging, and observability.

Until now, the recommended pattern for distributing periodic work across multiple instances of the same service has been `runutil.NewDistributedRepeat`, which leases a Redis (or kvrocks) key as a distributed lock. That approach forces every project that has periodic work to also operate a Redis-compatible store, even when the rest of the project only needs Postgres.

Replacing `DistributedRepeat` with a River periodic job removes that requirement: River serializes periodic execution through Postgres rows the project already owns. Projects that adopt riverutil purely for periodic jobs can drop their kvrocks/redis dependency entirely.

River also gives us durable retries, a job inspector UI at `/riverui/`, and atomic enqueueing inside `pgutil.Tx` — none of which `DistributedRepeat` provides.

## Hints

- See the `# Package pkg/riverutil` section in `docs.md` for the full wire-up.
- A periodic River job and a `DistributedRepeat`-wrapped worker are interchangeable for the "run this every N minutes, but only on one instance" use case. Migrate one worker at a time.
- Keep `runutil.ProvideWorker` for in-process work that does not need durability or distribution (e.g. the HTTP server itself, in-memory caches).
- Once the last `DistributedRepeat` is gone, the Redis client and its `cmd/root.go` provider can be removed. Check for other Redis users (rate limiting, sessions, caches) before deleting.
- River requires Postgres ≥ 12 and uses its own schema. `riverutil.Migrate` runs alongside `pgutil.Migrate` on startup.

## Examples

Before — periodic work via `DistributedRepeat` (requires Redis):

```go
func (w *DataSyncWorker) Workers() []runutil.Worker {
	return []runutil.Worker{
		runutil.DeclarativeWorker{
			Name: "DataSyncWorker",
			Worker: runutil.NewDistributedRepeat(
				w.redisClient, "data-sync-lock", 5*time.Minute,
				runutil.RetryJob(
					runutil.JobFunc(w.syncData),
					runutil.ExponentialBackoff{Initial: time.Minute, Max: 5 * time.Minute},
				),
			),
		},
	}
}
```

After — the same work as a River periodic job:

```go
type DataSyncArgs struct{}

func (DataSyncArgs) Kind() string { return "data_sync" }

type DataSync struct {
	river.WorkerDefaults[DataSyncArgs]
	pool *pgxpool.Pool
}

func NewDataSync(pool *pgxpool.Pool) *DataSync {
	return &DataSync{pool: pool}
}

func (w *DataSync) Config(config *river.Config) error {
	river.AddWorker(config.Workers, w)

	config.PeriodicJobs = append(config.PeriodicJobs, river.NewPeriodicJob(
		river.PeriodicInterval(5*time.Minute),
		func() (river.JobArgs, *river.InsertOpts) { return DataSyncArgs{}, nil },
		&river.PeriodicJobOpts{RunOnStart: true},
	))

	return nil
}

func (w *DataSync) Work(ctx context.Context, job *river.Job[DataSyncArgs]) error {
	return pgutil.Tx(ctx, w.pool, w.syncData)
}
```

Wire-up in `cmd/server.go`:

```go
riverutil.Provide(c, river_workers.NewDataSync),
```

River retries failed jobs automatically with exponential backoff, so the explicit `RetryJob` wrapper is not needed.
