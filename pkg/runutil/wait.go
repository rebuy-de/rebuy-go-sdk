package runutil

import (
	"context"
	"time"
)

// Wait is similar to [time.Sleep], but stops blocking when the context gets
// cancelled.
func Wait(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(d):
		return
	}
}
