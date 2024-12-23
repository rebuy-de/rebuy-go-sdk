package runutil

import (
	"context"

	"go.uber.org/dig"
)

// WorkerConfiger is for Workers that configure themselfes. This means they can define repeats, backoff and jitter themselves.
//
//	func (w *CommitFetcher) Workers() []runutil.Worker {
//		return []runutil.Worker{
//			runutil.DeclarativeWorker{
//				Name:   "Commits",
//				Worker: runutil.Repeat(5*time.Second, runutil.JobFunc(w.fetchCommits)),
//				Retry: runutil.ExponentialBackoff{
//					Initial:          time.Second,
//					Max:              time.Minute,
//					JitterProportion: 0.5,
//				},
//			},
//			runutil.DeclarativeWorker{
//				Name:   "PRs",
//				Worker: runutil.Repeat(5*time.Second, runutil.JobFunc(w.fetchPRs)),
//				Retry: runutil.ExponentialBackoff{
//					Initial:          time.Second,
//					Max:              time.Minute,
//					JitterProportion: 0.5,
//				},
//			},
//		}
//	}
type WorkerConfiger interface {
	Workers() []Worker
}

// WorkerGroup is a input parameter struct for Dig to retrieve all instances
// that implement the WorkerConfigerer.
type WorkerGroup struct {
	dig.In
	All []WorkerConfiger `group:"worker"`
}

// ProvideWorker injects a WorkerConfiger, which can later be started with
// RunProvidedWorkers.
func ProvideWorker(c *dig.Container, fn any) error {
	return c.Provide(fn, dig.Group("worker"), dig.As(new(WorkerConfiger)))
}

// RunProvidedWorkers starts all workers there were injected using
// RunAllWorkers.
func RunProvidedWorkers(ctx context.Context, c *dig.Container) error {
	return c.Invoke(func(in WorkerGroup) error {
		workers := []Worker{}
		for _, c := range in.All {
			if c == nil {
				continue
			}

			for _, w := range c.Workers() {
				workers = append(workers,
					NamedWorkerFromType(w, c),
				)
			}
		}
		return RunAllWorkers(ctx, workers...)
	})
}
