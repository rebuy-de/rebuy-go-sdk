package digutil

import "go.uber.org/dig"

type Optional[T any] struct {
	dig.In
	Value *T `optional:"true"`
}

func ProvideValue[T any](c *dig.Container, v T) error {
	return c.Provide(func() T {
		return v
	})
}
