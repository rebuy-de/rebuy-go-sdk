package riverutil

import (
	"github.com/riverqueue/river"
	"go.uber.org/dig"
)

type Configer interface {
	Config(*river.Config) error
}

type ConfigGroup struct {
	dig.In
	All []Configer `group:"river_configer"`
}

func Provide(c *dig.Container, fn any) error {
	return c.Provide(fn, dig.Group("river_configer"), dig.As(new(Configer)))
}
