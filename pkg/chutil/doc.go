// Package chutil provides a generic, runutil-compatible worker that
// batches rows and bulk-inserts them into ClickHouse.
//
// New opens a configured connection and wraps it in a Batcher. Connection and
// batcher defaults can be overridden with the With* options:
//
//	b, err := chutil.New[Row](
//		addr, auth, "INSERT INTO db.table (col_a, col_b)",
//		chutil.WithMaxSize(5000),
//	)
//
// To wire it into a dig container, register the address and credentials, then
// provide the batcher with Provide. Provide also registers the batcher as a
// worker, so its Run loop executes without any extra wiring. If ClickHouse is
// unreachable at startup, Provide logs a warning and supplies a nil *Batcher
// rather than returning an error, so the server starts regardless:
//
//	digutil.ProvideValue[chutil.Addr](c, "clickhouse:9000")
//	digutil.ProvideValue(c, auth) // chutil.Auth, e.g. from vaultutil.DecodeSecret
//	chutil.Provide[Row](c, "INSERT INTO db.table (col_a, col_b)")
//	runutil.ProvideWorker(c, func(b *chutil.Batcher[Row]) *chutil.Batcher[Row] { return b })
//
// # Migrations
//
// Migrate manages the ClickHouse schema, mirroring pgutil.Migrate. It creates
// the database named in Auth.Database (ClickHouse's namespace, analogous to a
// Postgres schema), runs versioned `*.up.sql` migrations via golang-migrate,
// then applies repeatable `R_*.sql` migrations tracked by content hash. Both
// kinds live under a top-level "migrations" directory in the embedded FS. Reuse
// the Addr and Auth already registered for the Batcher and invoke it directly:
//
//	digutil.ProvideValue[chutil.MigrationFS](c, chutil.MigrationFS(migrationsFS))
//	c.Invoke(chutil.Migrate)
//
// ClickHouse has no DDL transactions and the std driver runs one statement per
// call, so each R_*.sql file must contain a single statement, typically a
// `CREATE OR REPLACE VIEW`.
package chutil
