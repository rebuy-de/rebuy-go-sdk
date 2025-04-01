package workers

import (
	"context"
	"math/rand"
	"time"

	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/runutil"
)

// PeriodicTaskWorker is a worker that performs a variety of periodic tasks
type PeriodicTaskWorker struct {}

// NewPeriodicTaskWorker creates a new periodic task worker
func NewPeriodicTaskWorker() *PeriodicTaskWorker {
	return &PeriodicTaskWorker{}
}

// Workers implements the runutil.WorkerConfiger interface
func (w *PeriodicTaskWorker) Workers() []runutil.Worker {
	return []runutil.Worker{
		runutil.DeclarativeWorker{
			Name:   "MinuteTask",
			Worker: runutil.Repeat(1*time.Minute, runutil.JobFunc(w.minuteTask)),
		},
		runutil.DeclarativeWorker{
			Name:   "FiveMinuteTask",
			Worker: runutil.Repeat(5*time.Minute, runutil.JobFunc(w.fiveMinuteTask)),
		},
	}
}

// minuteTask is executed every minute
func (w *PeriodicTaskWorker) minuteTask(ctx context.Context) error {
	logutil.Get(ctx).Info("Running minute task")
	
	// Simulate some work with random duration
	time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)
	
	return nil
}

// fiveMinuteTask is executed every five minutes
func (w *PeriodicTaskWorker) fiveMinuteTask(ctx context.Context) error {
	logutil.Get(ctx).Info("Running five minute task")
	
	// Simulate some work with random duration
	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)
	
	return nil
}
