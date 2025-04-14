// Package runutil provides utilities for managing long-running services (Workers),
// one-off tasks (Jobs), and retry mechanisms with backoff strategies.
//
// # Worker Management with runutil
//
// The package provides a robust worker management system. This makes it easy to run and manage
// long-running services and one-off jobs.
//
// ## Worker Interface
//
//	// Worker is a service that is supposed to run continuously until the context is cancelled
//	type Worker interface {
//	    Run(ctx context.Context) error
//	}
//
//	// Job is a function that runs once and exits afterwards
//	type Job interface {
//	    RunOnce(ctx context.Context) error
//	}
//
// ## Worker with Dependency Injection
//
// The package integrates with the dig dependency injection library:
//
//	func SetupWorkers(ctx context.Context, c *dig.Container) error {
//	    // Register workers with the dig container
//	    err := errors.Join(
//	        runutil.ProvideWorker(c, workers.NewDatabaseCleanupWorker),
//	        runutil.ProvideWorker(c, workers.NewDataFetchWorker),
//	        runutil.ProvideWorker(c, workers.NewEventWatcherWorker),
//	    )
//	    if err != nil {
//	        return err
//	    }
//
//	    // Run all provided workers
//	    return runutil.RunProvidedWorkers(ctx, c)
//	}
//
// ## Retry and Backoff
//
// The runutil package provides utilities for retrying operations with backoff:
//
//	func FetchData(ctx context.Context) error {
//	    return runutil.Retry(ctx, func(ctx context.Context) error {
//	        // Operation that might fail
//	        return apiClient.FetchData(ctx)
//	    },
//	        runutil.WithMaxAttempts(5),
//	        runutil.WithBackoff(runutil.ExponentialBackoff(time.Second, 30*time.Second)),
//	        runutil.WithRetryableErrors(ErrTemporary, ErrTimeout),
//	    )
//	}
package runutil