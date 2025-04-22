package runutil

import (
	"context"
	"time"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type jobWorker struct {
	wait             time.Duration
	job              Job
	startImmediately bool
}

// Repeat reruns a job indefinitely until the context gets cancelled. The job
// will run at most once in the given time interval. This means the wait
// duration is not the sleep between executions, but the time between the start
// of runs (based on [time.Ticker]).
func Repeat(wait time.Duration, job Job, opts ...RepeatOption) Worker {
	w := &jobWorker{
		wait: wait,
		job:  job,
	}

	for _, o := range opts {
		o(w)
	}

	return w
}

type RepeatOption func(*jobWorker)

func WithStartImmediately() RepeatOption {
	return func(w *jobWorker) {
		w.startImmediately = true
	}
}

func (w jobWorker) Run(ctx context.Context) error {
	if w.startImmediately {
		err := w.runOnce(ctx)
		if err != nil {
			return err
		}
	}

	ticker := time.NewTicker(w.wait)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := w.runOnce(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (w jobWorker) runOnce(ctx context.Context) error {
	span, ctx := tracer.StartSpanFromContext(
		ctx, "runutil.job",
		tracer.Tag(ext.SpanKind, ext.SpanKindInternal),
		tracer.Tag(ext.ResourceName, logutil.GetSubsystem(ctx)),
	)
	err := w.job.RunOnce(ctx)
	if err != nil {
		span.Finish(tracer.WithError(err))
		return err
	} else {
		span.Finish()
	}

	return nil
}
