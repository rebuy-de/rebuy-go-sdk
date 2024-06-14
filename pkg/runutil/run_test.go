package runutil

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunAllWorkersExitedPrematurely(t *testing.T) {
	ctx := context.Background()

	err := RunAllWorkers(ctx,
		WorkerFunc(func(ctx context.Context) error {
			return nil
		}),
		WorkerFunc(func(ctx context.Context) error {
			return nil
		}),
		WorkerFunc(func(ctx context.Context) error {
			return nil
		}),
	)

	require.ErrorIs(t, err, ErrWorkerExitedPrematurely)
}

func TestRunnAllWorkersNoErrorOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// This waitgroup makes sure all go routines are started before cancelling
	// the context.
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		wg.Wait()
		cancel()
	}()

	err := RunAllWorkers(ctx,
		WorkerFunc(func(ctx context.Context) error {
			wg.Done()
			<-ctx.Done()
			return nil
		}),
		WorkerFunc(func(ctx context.Context) error {
			wg.Done()
			<-ctx.Done()
			return nil
		}),
		WorkerFunc(func(ctx context.Context) error {
			wg.Done()
			<-ctx.Done()
			return nil
		}),
	)

	require.NoError(t, err)
}

func TestRunnAllWorkersPassthroughErrorsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// This waitgroup makes sure all go routines are started before cancelling
	// the context.
	var wg sync.WaitGroup
	wg.Add(3)

	var omg = errors.New("some error")

	go func() {
		wg.Wait()
		cancel()
	}()

	err := RunAllWorkers(ctx,
		WorkerFunc(func(ctx context.Context) error {
			wg.Done()
			<-ctx.Done()
			return nil
		}),
		WorkerFunc(func(ctx context.Context) error {
			wg.Done()
			<-ctx.Done()
			return omg
		}),
		WorkerFunc(func(ctx context.Context) error {
			wg.Done()
			<-ctx.Done()
			return nil
		}),
	)

	require.ErrorIs(t, err, omg)
}

func TestRunnAllWorkersPassthroughErrors(t *testing.T) {
	ctx := context.Background()

	var omg = errors.New("some error")

	err := RunAllWorkers(ctx,
		WorkerFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}),
		WorkerFunc(func(ctx context.Context) error {
			return omg
		}),
		WorkerFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}),
	)

	require.ErrorIs(t, err, omg)
}
