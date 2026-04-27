package riverutil

import (
	"context"
	"errors"

	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/logutil"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type logutilMiddleware struct {
}

func (*logutilMiddleware) IsMiddleware() bool {
	return true
}

func (*logutilMiddleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(ctx context.Context) error) error {
	ctx = logutil.Start(ctx, "river_job")

	ctx = logutil.WithFields(ctx, map[string]any{
		"river-kind":   job.Kind,
		"river-job-id": job.ID,
		"river-args":   string(job.EncodedArgs),
	})

	if job.Attempt > 1 {
		ctx = logutil.WithField(ctx, "river-attempts", job.Attempt)
	}

	logutil.Get(ctx).Info("starting river job")

	err := doInner(ctx)

	switch {
	case errors.Is(err, new(river.JobCancelError)):
		logutil.Get(ctx).Warn("river job cancelled", "error", err)
	case errors.Is(err, new(river.JobSnoozeError)):
		logutil.Get(ctx).Info("river job snoozed", "error", err)
	case err != nil:
		logutil.Get(ctx).Error("river job failed", "error", err)
	default:
		logutil.Get(ctx).Debug("river job succeeded")
	}

	return err
}
