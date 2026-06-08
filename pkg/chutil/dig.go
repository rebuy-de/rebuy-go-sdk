package chutil

import (
	"go.uber.org/dig"
)

// Provide registers a *Batcher[T] built from an Addr and Auth already present in
// the container. Register the address and credentials with digutil.ProvideValue,
// and register the returned batcher as a worker (e.g. via runutil.ProvideWorker
// wrapping a runutil.DeclarativeWorker) so its Run loop is executed.
//
// A generic function cannot be passed to dig.Provide directly; this helper pins
// the concrete row type T inside a closure.
func Provide[T any](c *dig.Container, insertSQL string, opts ...Option) error {
	return c.Provide(func(addr Addr, auth Auth) (*Batcher[T], error) {
		return New[T](addr, auth, insertSQL, opts...)
	})
}
