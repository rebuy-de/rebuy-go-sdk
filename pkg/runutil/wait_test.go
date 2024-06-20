package runutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWait(t *testing.T) {
	stopwatch := time.Now()
	for i := 0; i < 4; i++ {
		Wait(context.Background(), 10*time.Millisecond)
	}
	duration := time.Since(stopwatch)

	require.Greater(t, duration, 40*time.Millisecond)
}

func TestWaitCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stopwatch := time.Now()
	Wait(ctx, 10*time.Second)
	duration := time.Since(stopwatch)

	require.Less(t, duration, time.Second)
}
