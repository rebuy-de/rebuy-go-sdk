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

- Each project has its own variant of `pkg/dal/sqlc/tx.go`. boombot exposes `*sqlc.DB`, `*sqlc.Tx` (with embedded `*Queries` and a `.Tx()` accessor), and a `RiverTxWorkFunc` wrapper — the strongest pattern. platform-inventory has `Tx` and `Hijack` but no worker wrapper. update-manager has a stripped-down wrapper that passes `*Queries` to the closure with no `.Tx()` accessor, which makes atomic enqueue via `riverutil.Insert` impossible.
- Raw `river.UniqueOpts{ByArgs: true}` at insert sites. River's default `ByState` includes `Completed`, which blocks legitimate re-insertion (e.g. an alert pipeline that needs to run again when an alert resolves).
- `Args` structs missing JSON tags. River computes `unique_key` by JSON-serializing Args; untagged fields are not reliably included and dedup silently misbehaves.
- DB writes followed by a non-transactional `river.ClientFromContext[pgx.Tx](ctx).Insert(...)`. If the worker crashes between the write and the enqueue, the DB state advances but the follow-up job never runs.
- Webhook handler enqueues without `UniqueOpts`. External systems retry webhooks; without dedup, retries produce duplicate jobs.
- External API calls (GitHub, Slack) held inside a `pgutil.Tx` — the pooled connection is blocked for the duration of the call and any failure rolls back the DB writes.
- `context.WithoutCancel` sprinkled through workers without comments. It is a meaningful choice (preserve work on shutdown vs. respect deadlines) and undocumented use makes it impossible to tell deliberate from accidental.

The SDK does not ship a worker-transaction wrapper because forcing `pgx.Tx` at every call site loses the ergonomics of a project-local `*sqlc.Tx` (which can embed the generated `*Queries` directly). Instead, the `# Package pkg/riverutil` → `## Work Method Style` section in `docs.md` publishes a canonical `pkg/dal/sqlc/tx.go` that each project should adopt verbatim. The SDK does ship `pgutil.Tx`, `pgutil.Hijack`, `riverutil.Insert`, and `riverutil.UniqueOptsByArgs` as primitives — the canonical `tx.go` is built on top of those.

## Hints

- Adopt the canonical `pkg/dal/sqlc/tx.go` from `docs.md` (the `# Package pkg/riverutil` → `## Work Method Style` section). Existing projects: diff against the canonical and converge. boombot is closest already (only minor refinements). platform-inventory has `Tx`/`Hijack` but is missing `RiverTxWorkFunc`. update-manager needs the full triple — `*sqlc.DB`, `*sqlc.Tx`, `.Tx()` accessor, `RiverTxWorkFunc`.
- Audit every `river.UniqueOpts{ByArgs: true}` and replace with `riverutil.UniqueOptsByArgs`. Grep for both `UniqueOpts{` and `ByArgs:` to find them all.
- Audit every `Args` struct for JSON tags before relying on uniqueness. Untagged fields break dedup silently — they do not error.
- In workers with preceding DB writes, switch the worker registration to `sqlc.RiverTxWorkFunc(w.db, w.work)` and use `riverutil.Insert(ctx, qtx.Tx(), ...)` for follow-up jobs. This is the only way the writes and the enqueue commit atomically. Stop using `river.ClientFromContext[pgx.Tx](ctx).Insert(...)` from inside such workers.
- HTTP/webhook handler enqueues should pass `UniqueOpts: riverutil.UniqueOptsByArgs` too, especially when the trigger is an external system that retries on its own (GitHub webhooks, Slack interactions, alerting platforms).
- Lift external API calls out of transactions. If the call must follow a DB write, schedule it via a follow-up river job rather than calling it inside `db.Tx` / `RiverTxWorkFunc` — the job inherits durability and retries.
- Audit `context.WithoutCancel` usages. Remove the unintentional ones; comment the deliberate ones with a one-line explanation of why cancellation is being dropped. Inside a river `Work` method, never wrap the ctx River passes in.

## Examples

Before — stripped `tx.go` (no `*Tx` / `.Tx()`), raw `UniqueOpts`, non-atomic enqueue, missing JSON tag:

```go
// pkg/dal/sqlc/tx.go (update-manager-style stripped wrapper)
func (db *DB) Tx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(db.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// pkg/app/river_workers/process_thing.go
func (w *ProcessThing) Work(ctx context.Context, job *river.Job[ProcessThingArgs]) error {
	err := w.db.UpdateThing(ctx, job.Args.ID) // outside any tx
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

After — canonical `tx.go`, `RiverTxWorkFunc` registration, atomic enqueue, JSON-tagged Args:

```go
// pkg/dal/sqlc/tx.go — see docs.md for the full canonical version.
// DB / Tx / RiverTxWorkFunc, all three.

// pkg/app/river_workers/process_thing.go
func (w *ProcessThing) Config(config *river.Config) error {
	river.AddWorker(config.Workers, sqlc.RiverTxWorkFunc(w.db, w.work))
	return nil
}

func (w *ProcessThing) work(ctx context.Context, qtx *sqlc.Tx, job *river.Job[ProcessThingArgs]) error {
	err := qtx.UpdateThing(ctx, job.Args.ID) // qtx embeds *Queries directly
	if err != nil {
		return err
	}

	return riverutil.Insert(ctx, qtx.Tx(), FollowupArgs{ID: job.Args.ID}, &river.InsertOpts{
		UniqueOpts: riverutil.UniqueOptsByArgs,
	})
}

type FollowupArgs struct {
	ID uuid.UUID `json:"id"`
}

func (FollowupArgs) Kind() string { return "followup" }
```

The work runs in a single transaction; the follow-up job and the DB update commit (or roll back) together. Dedup uses `riverutil.UniqueOptsByArgs`, which excludes `Completed` from `ByState` so the same args can be re-inserted after the job finishes.
