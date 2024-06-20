package runutil

import (
	"context"
	"errors"
	"sync"
)

// WorkerExitedPrematurely indicates that a worker exited in [RunAllWorkers]
// while the context was not cancelled yet.
var ErrWorkerExitedPrematurely = errors.New("worker exited prematurely")

// RunAllWorkers starts all workers in goroutines and waits until all are
// exited.
//
// Behaviour:
//   - The execution for all workers get cancelled when the first worker
//     exists, regardless of the exit code.
//   - Err is nil, if the context gets cancelled and the workers return a nil
//     error too.
//   - Err contains [WorkerExitedPrematurely], if the workers return a nil error
//     while the context was not cancelled.
//   - Err contains all errors, returned by the workers.
func RunAllWorkers(ctx context.Context, workers ...Worker) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(workers))

	var errs collector[error]

	for _, w := range workers {
		w := w
		go func() {
			defer wg.Done()
			defer cancel()
			err := w.Run(ctx)
			if err != nil {
				errs.Append(err)
			} else if ctx.Err() == nil {
				// It means that the works exited itself, if the worker returns
				// nil and the context was not cancelled yet.
				errs.Append(ErrWorkerExitedPrematurely)
			}
			// Otherwise the Context was cancelled and the worker did not
			// return an error, which means it shut down gracefully.
		}()
	}

	wg.Wait()

	return errors.Join(errs.Result()...)
}

// RunAllJobs runs all jobs in parallel and return their errors.
func RunAllJobs(ctx context.Context, jobs ...Job) error {
	var wg sync.WaitGroup
	wg.Add(len(jobs))

	var errs collector[error]

	for _, j := range jobs {
		j := j
		go func() {
			defer wg.Done()
			err := j.RunOnce(ctx)
			if err != nil {
				errs.Append(err)
			}
		}()
	}

	wg.Wait()

	return errors.Join(errs.Result()...)
}

// collector is a helper type for a concurrency-safe append to slices.
type collector[T any] struct {
	result []T
	mux    sync.Mutex
}

func (c *collector[T]) Append(value T) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.result = append(c.result, value)
}

func (c *collector[T]) Result() []T {
	return c.result
}
