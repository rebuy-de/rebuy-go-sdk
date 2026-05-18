---
id: M0011
title: Align transaction and river patterns
date: 2026-05-18
sdk_version: v10
type: minor
---

# Align transaction and river patterns

## Reasoning

An audit of projects that adopted `pkg/pgutil` and `pkg/riverutil` found the transaction-around-river-enqueue pattern drifting in several directions. Recurring problems:

- Local copies of a `RiverTxWorkFunc` helper, each slightly different.
- Raw `river.UniqueOpts{ByArgs: true}` at insert sites — River's default `ByState` includes `Completed`, which blocks legitimate re-insertion (e.g. an alert pipeline that needs to run again when an alert resolves).
- `Args` structs missing JSON tags. River computes `unique_key` by JSON-serializing Args; untagged fields are not reliably included and dedup silently misbehaves.
- DB writes followed by a non-transactional `river.ClientFromContext[pgx.Tx](ctx).Insert(...)`. If the worker crashes between the write and the enqueue, the DB state advances but the follow-up job never runs.
- Webhook handler enqueues without `UniqueOpts`. External systems retry webhooks; without dedup, retries produce duplicate jobs.
- External API calls (GitHub, Slack) held inside a `pgutil.Tx` — the pooled connection is blocked for the duration of the call and any failure rolls back the DB writes.
- `context.WithoutCancel` sprinkled through workers without comments. It is a meaningful choice (preserve work on shutdown vs. respect deadlines) and undocumented use makes it impossible to tell deliberate from accidental.

The SDK now ships `riverutil.TxWorkFunc`, which standardizes the wrap-work-in-tx-and-call-JobCompleteTx pattern. The docs at `# Package pkg/pgutil` (transactions, external calls, `context.WithoutCancel`) and `# Package pkg/riverutil` (work method style, enqueueing jobs) tighten the surrounding guidance. This migration is the per-project rollout.

## Hints

- Replace any project-local `RiverTxWorkFunc` (or equivalent) with `riverutil.TxWorkFunc`. Projects that need a richer transaction parameter — e.g. an sqlc `*Tx` with a `.Tx()` accessor — keep a 4-line local wrapper that composes over the SDK helper. Everything else in the local `tx.go` can be deleted.
- Audit every `river.UniqueOpts{ByArgs: true}` and replace with `riverutil.UniqueOptsByArgs`. Grep for both `UniqueOpts{` and `ByArgs:` to find them all.
- Audit every `Args` struct for JSON tags before relying on uniqueness. Untagged fields break dedup silently — they do not error.
- In workers with preceding DB writes, replace `river.ClientFromContext[pgx.Tx](ctx).Insert(...)` with `riverutil.Insert(ctx, tx, ...)` and wrap the body in `pgutil.Tx` or `riverutil.TxWorkFunc`. This is the only way the writes and the enqueue commit atomically.
- HTTP/webhook handler enqueues should pass `UniqueOpts: riverutil.UniqueOptsByArgs` too, especially when the trigger is an external system that retries on its own (GitHub webhooks, Slack interactions, alerting platforms).
- Lift external API calls out of transactions. If the call must follow a DB write, schedule it via a follow-up river job rather than calling it inside the tx — the job inherits durability and retries.
- Audit `context.WithoutCancel` usages. Remove the unintentional ones; comment the deliberate ones with a one-line explanation of why cancellation is being dropped. Inside a river `Work` method, never wrap the ctx River passes in.

## Examples

Before — project-local helper, raw `UniqueOpts`, non-atomic enqueue:

```go
// pkg/dal/sqlc/tx.go (local copy)
func RiverTxWorkFunc[T river.JobArgs](db *DB, fn func(context.Context, *Tx, *river.Job[T]) error) river.Worker[T] {
	return river.WorkFunc(func(ctx context.Context, job *river.Job[T]) error {
		return db.Tx(ctx, func(qtx *Tx) error {
			err := fn(ctx, qtx, job)
			if err != nil {
				return err
			}
			_, err = river.JobCompleteTx[*riverpgxv5.Driver](ctx, qtx.Tx(), job)
			return err
		})
	})
}

// pkg/app/river_workers/process_thing.go
func (w *ProcessThing) Work(ctx context.Context, job *river.Job[ProcessThingArgs]) error {
	err := w.queries.UpdateThing(ctx, job.Args.ID) // outside any tx
	if err != nil {
		return err
	}

	client := river.ClientFromContext[pgx.Tx](ctx)
	_, err = client.Insert(ctx, FollowupArgs{ID: job.Args.ID}, &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true}, // blocks re-insert after Completed
	})
	return err
}

type FollowupArgs struct {
	ID uuid.UUID // missing JSON tag — silent dedup bug
}
```

After — SDK helper, `UniqueOptsByArgs`, atomic enqueue, JSON-tagged Args:

```go
// pkg/app/river_workers/process_thing.go
func (w *ProcessThing) Config(config *river.Config) error {
	river.AddWorker(config.Workers, riverutil.TxWorkFunc(w.pool, w.work))
	return nil
}

func (w *ProcessThing) work(ctx context.Context, tx pgx.Tx, job *river.Job[ProcessThingArgs]) error {
	qtx := w.queries.WithTx(tx)
	err := qtx.UpdateThing(ctx, job.Args.ID)
	if err != nil {
		return err
	}

	return riverutil.Insert(ctx, tx, FollowupArgs{ID: job.Args.ID}, &river.InsertOpts{
		UniqueOpts: riverutil.UniqueOptsByArgs,
	})
}

type FollowupArgs struct {
	ID uuid.UUID `json:"id"`
}

func (FollowupArgs) Kind() string { return "followup" }
```

The local `RiverTxWorkFunc` and the surrounding `tx.go` boilerplate are gone. The work runs in a single transaction; the follow-up job and the DB update commit (or roll back) together. Dedup uses the SDK's `UniqueOptsByArgs`, which excludes `Completed` from `ByState` so the same args can be re-inserted after the job finishes.
