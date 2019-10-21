package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/executil"
)

type Executor struct {
	ctx context.Context
	err error
}

func NewExecutor(ctx context.Context) *Executor {
	return &Executor{
		ctx: ctx,
	}
}

func (e *Executor) Err() error {
	return e.err
}

func (e *Executor) GetString(command string, args ...string) string {
	if e.ctx.Err() != nil {
		return ""
	}

	if e.err != nil {
		return ""
	}

	c := exec.Command(command, args...)
	out := bytes.Buffer{}
	c.Stdout = &out
	c.Stderr = os.Stderr
	e.err = executil.Run(e.ctx, c)
	return strings.TrimSpace(out.String())
}

func (e *Executor) GetInt64(command string, args ...string) int64 {
	if e.ctx.Err() != nil {
		return 0
	}

	if e.err != nil {
		return 0
	}

	var i int64
	i, e.err = strconv.ParseInt(e.GetString(command, args...), 10, 64)
	return i
}
