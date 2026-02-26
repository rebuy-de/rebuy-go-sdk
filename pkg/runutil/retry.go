package runutil

import (
	"context"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
)

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
