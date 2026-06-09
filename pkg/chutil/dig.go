package chutil

import (
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/runutil"
	"go.uber.org/dig"
)

// Provide registers a *Batcher[T] built from an Addr and Auth already present in
// the container, and wires it up as a worker so its Run loop is executed. Just
// register the address and credentials with digutil.ProvideValue beforehand;
// the worker registration no longer has to be done by the application.
//
// A generic function cannot be passed to dig.Provide directly; this helper pins
// the concrete row type T inside a closure.
func Provide[T any](c *dig.Container, insertSQL string, opts ...Option) error {
	err := c.Provide(func(addr Addr, auth Auth) (*Batcher[T], error) {
		return New[T](addr, auth, insertSQL, opts...)
	})
	if err != nil {
		return err
	}

	return runutil.ProvideWorker(c, func(b *Batcher[T]) *Batcher[T] { return b })
}
