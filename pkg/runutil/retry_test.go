package runutil

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRetryJobSucceedsFirstAttempt(t *testing.T) {
	var calls int
	job := RetryJob(JobFunc(func(ctx context.Context) error {
		calls++
		return nil
	}), StaticBackoff{Sleep: time.Second})

	err := job.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryJobSucceedsAfterFailures(t *testing.T) {
	var calls int
	job := RetryJob(JobFunc(func(ctx context.Context) error {
		calls++
		if calls < 4 {
			return fmt.Errorf("attempt %d failed", calls)
		}
		return nil
	}), StaticBackoff{Sleep: time.Millisecond})

	err := job.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if calls != 4 {
		t.Fatalf("expected 4 calls, got %d", calls)
	}
}

func TestRetryJobContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var calls int
	job := RetryJob(JobFunc(func(ctx context.Context) error {
		calls++
		if calls >= 2 {
			cancel()
		}
		return fmt.Errorf("always fails")
	}), StaticBackoff{Sleep: time.Millisecond})

	err := job.RunOnce(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRetryJobBackoffApplied(t *testing.T) {
	var calls int
	backoff := StaticBackoff{Sleep: 50 * time.Millisecond}

	job := RetryJob(JobFunc(func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return fmt.Errorf("fail")
		}
		return nil
	}), backoff)

	start := time.Now()
	err := job.RunOnce(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}

	// 2 failures = 2 backoff waits of 50ms each = ~100ms minimum
	if elapsed < 90*time.Millisecond {
		t.Fatalf("expected at least 90ms elapsed, got %v", elapsed)
	}
}
