package riverutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/logutil"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// ActiveJobStates excludes Completed state so jobs can be re-inserted
// after previous jobs with the same args have finished.
var ActiveJobStates = []rivertype.JobState{
	rivertype.JobStateAvailable,
	rivertype.JobStatePending,
	rivertype.JobStateRunning,
	rivertype.JobStateRetryable,
	rivertype.JobStateScheduled,
}

// UniqueOptsByArgs provides standard unique options for jobs that should
// be deduplicated by args, but can be re-inserted after completion.
var UniqueOptsByArgs = river.UniqueOpts{
	ByArgs:  true,
	ByState: ActiveJobStates,
}

func Insert(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) error {
	response, err := river.ClientFromContext[pgx.Tx](ctx).InsertTx(ctx, tx, args, opts)
	if response.UniqueSkippedAsDuplicate {
		logutil.Get(ctx).Info("job was skipped as duplicate", "kind", args.Kind(), "args", args)
	}
	return err
}
