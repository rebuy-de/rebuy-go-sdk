package runutil

import (
	"context"
)

// DeclarativeWorker is an alternative to building the worker behaviour with
// chained functions.If automatically chains worker functions based on defined
// field in the most sensful order.
//
// It satisfies the Worker interface for easier use.
type DeclarativeWorker struct {
	Name   string
	Worker Worker
	Retry  Backoff
}

func (w DeclarativeWorker) Run(ctx context.Context) error {
	worker := w.Worker

	if w.Name != "" {
		worker = NamedWorker(worker, w.Name)
	}

	if w.Retry != nil {
		worker = Retry(worker, w.Retry)
	}

	return worker.Run(ctx)
}
