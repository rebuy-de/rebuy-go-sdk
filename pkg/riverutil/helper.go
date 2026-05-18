package riverutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/pgutil"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
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
	if err != nil {
		return err
	}
	if response.UniqueSkippedAsDuplicate {
		logutil.Get(ctx).Info("job was skipped as duplicate", "kind", args.Kind(), "args", args)
	}
	return nil
}

// TxWorkFunc wraps fn so each invocation runs inside a pgutil.Tx and the
// job-completion row commits in the same transaction as fn's DB writes.
//
// fn must not call river.JobCompleteTx itself; the wrapper does so on a nil
// return. Returning river.JobCancel, river.JobSnooze, or any other error
// skips completion and rolls the transaction back — the cancel/snooze
// signal still reaches River because it is propagated through the returned
// error.
func TxWorkFunc[T river.JobArgs](
	pool *pgxpool.Pool,
	fn func(ctx context.Context, tx pgx.Tx, job *river.Job[T]) error,
) river.Worker[T] {
	return river.WorkFunc(func(ctx context.Context, job *river.Job[T]) error {
		return pgutil.Tx(ctx, pool, func(tx pgx.Tx) error {
			err := fn(ctx, tx, job)
			if err != nil {
				return err
			}

			_, err = river.JobCompleteTx[*riverpgxv5.Driver](ctx, tx, job)
			return err
		})
	})
}
