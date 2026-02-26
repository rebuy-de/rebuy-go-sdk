package cmdutil

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// SignalRootContext returns a new empty context, that gets cancelled on SIGINT
// or SIGTERM.
func SignalRootContext() context.Context {
	return SignalContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

// SignalContext returns a copy of the parent context that gets cancelled if
// the application gets any of the given signals.
func SignalContext(ctx context.Context, signals ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)

	go func() {
		sig := <-c
		slog.Debug("received signal", "signal", sig)
		cancel()

		sig = <-c
		slog.Debug("received signal", "signal", sig)
		slog.Error("Two interrupts received. Exiting immediately. Note that data loss may have occurred.")
		os.Exit(ExitCodeMultipleInterrupts)
	}()

	return ctx
}

type RunFunc func(cmd *cobra.Command, args []string)
type RunFuncWithContext func(ctx context.Context, cmd *cobra.Command, args []string)

// ContextWithDelay delays the context cancel by the given delay. In the
// background it creates a new context with ContextWithValuesFrom and cancels
// it after the original one got canceled.
func ContextWithDelay(in context.Context, delay time.Duration) context.Context {
	out := context.WithoutCancel(in)
	out, cancel := context.WithCancel(out)

	go func() {
		defer cancel()
		<-in.Done()
		time.Sleep(delay)
	}()
	return out
}
