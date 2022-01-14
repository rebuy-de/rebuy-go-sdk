package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/executil"
)

type ChainExecutor struct {
	ctx context.Context
	err error
}

func NewChainExecutor(ctx context.Context) *ChainExecutor {
	return &ChainExecutor{
		ctx: ctx,
	}
}

func (e *ChainExecutor) Err() error {
	return e.err
}

func (e *ChainExecutor) Run(command string, args ...string) {
	if e.ctx.Err() != nil {
		return
	}

	if e.err != nil {
		return
	}

	c := exec.Command(command, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	e.err = executil.Run(e.ctx, c)
}

func (e *ChainExecutor) OutputString(command string, args ...string) string {
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

func (e *ChainExecutor) OutputInt64(command string, args ...string) int64 {
	if e.ctx.Err() != nil {
		return 0
	}

	if e.err != nil {
		return 0
	}

	var i int64
	i, e.err = strconv.ParseInt(e.OutputString(command, args...), 10, 64)
	return i
}
