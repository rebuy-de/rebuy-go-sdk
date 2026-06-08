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
// To wire it into a dig container, register the address and credentials, provide
// the batcher with Provide, and register the same instance as a worker so its
// Run loop executes (Batcher satisfies runutil.WorkerConfiger):
//
//	digutil.ProvideValue[chutil.Addr](c, "clickhouse:9000")
//	digutil.ProvideValue(c, auth) // chutil.Auth, e.g. from vaultutil.DecodeSecret
//	chutil.Provide[Row](c, "INSERT INTO db.table (col_a, col_b)")
//	runutil.ProvideWorker(c, func(b *chutil.Batcher[Row]) *chutil.Batcher[Row] { return b })
package chutil
