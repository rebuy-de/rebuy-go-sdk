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
package chutil
