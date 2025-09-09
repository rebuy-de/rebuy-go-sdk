package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/runutil"
	"github.com/redis/go-redis/v9"
)

// DataSyncWorker is responsible for periodically syncing data
type DataSyncWorker struct {
	redisClient *redis.Client
}

// NewDataSyncWorker creates a new data sync worker
func NewDataSyncWorker(redisClient *redis.Client) *DataSyncWorker {
	return &DataSyncWorker{
		redisClient: redisClient,
	}
}

// Workers implements the runutil.WorkerConfiger interface
func (w *DataSyncWorker) Workers() []runutil.Worker {
	return []runutil.Worker{
		runutil.DeclarativeWorker{
			Name:   "DataSyncWorker",
			Worker: runutil.Repeat(5*time.Minute, runutil.JobFunc(w.syncData)),
		},
	}
}

// syncData performs the actual data synchronization
func (w *DataSyncWorker) syncData(ctx context.Context) error {
	logutil.Get(ctx).Info("Synchronizing data...")

	// Record the current time in Redis as our last sync
	_, err := w.redisClient.Set(ctx, "last_sync", time.Now().Format(time.RFC3339), 0).Result()
	if err != nil {
		return fmt.Errorf("failed to update last sync time: %w", err)
	}

	// Simulate some work
	time.Sleep(500 * time.Millisecond)

	// Update the counter in Redis
	_, err = w.redisClient.Incr(ctx, "sync_count").Result()
	if err != nil {
		return fmt.Errorf("failed to update sync counter: %w", err)
	}

	logutil.Get(ctx).Info("Data synchronization completed")
	return nil
}
