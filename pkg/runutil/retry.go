package runutil

import (
	"context"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
)

// RetryJob retries a Job with backoff until it succeeds or the context is
// cancelled. Unlike Retry, this wraps a single-execution Job and returns nil
// on success. This is useful for retrying inside a DistributedRepeat loop
// where the retry should happen while still holding the distributed lock.
func RetryJob(job Job, bo Backoff) Job {
	return JobFunc(func(ctx context.Context) error {
		var attempt int
		for ctx.Err() == nil {
			err := job.RunOnce(ctx)
			if err == nil {
				return nil
			}

			attempt++
			logutil.Get(ctx).Warn("job failed", "attempt", attempt, "error", err)

			Wait(ctx, bo.Duration(attempt))
		}

		return ctx.Err()
	})
}

// Retry restarts a Worker forever when it exists. This happens regardless of
// whether the worker returns an error or nil. The worker only stops with
// restarting, when the context gets cancelled.
func Retry(worker Worker, bo Backoff) Worker {
	return WorkerFunc(func(ctx context.Context) error {
		var attempt int
		for ctx.Err() == nil {
			Wait(ctx, bo.Duration(attempt))

			err := worker.Run(ctx)
			if err != nil {
				attempt += 1
				logutil.Get(ctx).Warn("worker failed", "attempt", attempt, "error", err)
			} else {
				attempt = 0
			}
		}

		return nil
	})
}
