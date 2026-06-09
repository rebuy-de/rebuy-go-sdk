package chutil

import (
	"log/slog"

	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/runutil"
	"go.uber.org/dig"
)

// Provide registers a *Batcher[T] built from an Addr and Auth already present in
// the container, and wires it up as a worker so its Run loop is executed. Just
// register the address and credentials with digutil.ProvideValue beforehand;
// the worker registration no longer has to be done by the application.
//
// On open failure Provide logs a warning and provides a nil *Batcher instead of
// returning an error, so an unreachable ClickHouse never blocks server startup.
// Consumers that nil-guard the batcher skip analytics silently; the worker loop
// is also skipped.
//
// A generic function cannot be passed to dig.Provide directly; this helper pins
// the concrete row type T inside a closure.
func Provide[T any](c *dig.Container, insertSQL string, opts ...Option) error {
	err := c.Provide(func(addr Addr, auth Auth) *Batcher[T] {
		b, err := New[T](addr, auth, insertSQL, opts...)
		if err != nil {
			slog.Warn("disabling clickhouse analytics: open connection failed", "error", err)
			return nil
		}
		return b
	})
	if err != nil {
		return err
	}

	return c.Provide(func(b *Batcher[T]) runutil.WorkerConfiger {
		if b == nil {
			return nil
		}
		return b
	}, dig.Group("worker"))
}
