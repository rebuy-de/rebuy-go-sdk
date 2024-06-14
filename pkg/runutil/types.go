package runutil

import "context"

// Worker is a service that is supposed to run continuously until the context
// gets cancelled.
type Worker interface {
	Run(ctx context.Context) error
}

// WorkerFunc is a helper to cast a function directly to a Worker.
type WorkerFunc func(ctx context.Context) error

func (fn WorkerFunc) Run(ctx context.Context) error {
	return fn(ctx)
}

// Job is a function that runs once and exits afterwards.
type Job interface {
	RunOnce(ctx context.Context) error
}

// JobFunc is a helper to cast a function directly to a Job.
type JobFunc func(ctx context.Context) error

func (fn JobFunc) RunOnce(ctx context.Context) error {
	return fn(ctx)
}
