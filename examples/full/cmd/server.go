package cmd

import (
	"context"
	"errors"

	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/pkg/app/handlers"
	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/pkg/app/templates"
	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/pkg/app/workers"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/runutil"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/webutil"
	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"
)

func RunServer(ctx context.Context, c *dig.Container) error {
	// Register core dependencies
	err := errors.Join(
		c.Provide(templates.New),

		// Register HTTP handlers
		webutil.ProvideHandler(c, handlers.NewIndexHandler),
		webutil.ProvideHandler(c, handlers.NewHealthHandler),
		webutil.ProvideHandler(c, handlers.NewUsersHandler),

		c.Provide(func(
			authMiddleware webutil.AuthMiddleware,
		) webutil.Middlewares {
			return webutil.Middlewares(append(
				webutil.DefaultMiddlewares(),
				authMiddleware,
			))
		}),

		// Register background workers
		runutil.ProvideWorker(c, func(redisClient *redis.Client) *workers.DataSyncWorker {
			return workers.NewDataSyncWorker(redisClient)
		}),
		runutil.ProvideWorker(c, workers.NewPeriodicTaskWorker),

		// Register the HTTP server itself
		runutil.ProvideWorker(c, webutil.NewServer),
	)
	if err != nil {
		return err
	}

	// Start all registered workers
	return runutil.RunProvidedWorkers(ctx, c)
}
