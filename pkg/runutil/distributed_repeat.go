package runutil

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
	"github.com/redis/go-redis/v9"
)

type DistributedRepeat struct {
	client   redis.UniversalClient
	name     string
	cooldown time.Duration
	job      Job
}

// NewDistributedRepeat creates a [runutil.Worker] from a [runutil.Job] similar to [runutil.Repeat]. The difference is
// that it uses Redis for coordinating the repeats so that the repeat cooldown is respected across multiple replicas.
//
// The Redis docs say that SetNX is not good enough for locking, [but so is their][1] suggested Redlock algorithm.
// Therefore we can simplify things by using SetNX directly. The lock does not have any correctness guarantees.
//
// The lock gets refreshed every tenth of the cooldown to prevent issues when the job takes longer than the cooldown.
//
// [1]: https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html
func NewDistributedRepeat(client redis.UniversalClient, name string, cooldown time.Duration, job Job) Worker {
	return &DistributedRepeat{
		client:   client,
		name:     name,
		cooldown: cooldown,
		job:      job,
	}
}

func (r *DistributedRepeat) Run(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("get hostname: %w", err)
	}

	for ctx.Err() == nil {
		err := r.attemptExecution(ctx, hostname)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *DistributedRepeat) attemptExecution(ctx context.Context, hostname string) error {
	ok, err := r.client.SetNX(ctx, r.name, hostname, r.cooldown).Result()
	if err != nil {
		return fmt.Errorf("setnx %#v for lock: %w", r.name, err)
	}

	if ok {
		ticker := time.NewTicker(r.cooldown / 10)
		done := make(chan struct{})
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Use a timeout context to prevent blocking indefinitely on cancelled parent context
					expireCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
					err := r.client.Expire(expireCtx, r.name, r.cooldown).Err()
					cancel()
					if err != nil {
						logutil.Get(ctx).Error("refreshing repeat lock", "error", err)
					}
				}
			}
		}()

		err := r.runOnce(ctx)
		close(done)
		wg.Wait()

		if err != nil {
			return err
		}
	}

	wait, err := r.client.TTL(ctx, r.name).Result()
	if err != nil {
		return fmt.Errorf("get ttl %#v for lock: %w", r.name, err)
	}

	// add jitter of 0% - 5% of total wait time
	jitter := time.Duration(float64(r.cooldown) / 20. * rand.Float64())

	logutil.Get(ctx).Debug("distributed sleep", "wait", wait, "jitter", jitter, "total", wait+jitter)

	select {
	case <-time.After(wait + jitter):
	case <-ctx.Done():
	}

	return nil
}

func (r *DistributedRepeat) runOnce(ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(
		ctx, "runutil.job",
		tracer.Tag(ext.SpanKind, ext.SpanKindInternal),
		tracer.Tag(ext.ResourceName, logutil.GetSubsystem(ctx)),
	)
	err := r.job.RunOnce(ctx)
	HealthCheckpoint(ctx, err)

	if err != nil {
		span.Finish(tracer.WithError(err))
		return err
	} else {
		span.Finish()
	}

	return nil
}
