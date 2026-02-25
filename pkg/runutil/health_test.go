package runutil

import (
	"context"
	"errors"
	"testing"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
	"github.com/stretchr/testify/assert"
)

func TestHealthRegistry_ReturnsSameMonitor(t *testing.T) {
	r := &healthRegistryImpl{
		monitors: map[string]*healthMonitor{},
	}

	m1 := r.get("test-worker")
	m2 := r.get("test-worker")

	assert.Same(t, m1, m2)
}

func TestHealthRegistry_DifferentNames(t *testing.T) {
	r := &healthRegistryImpl{
		monitors: map[string]*healthMonitor{},
	}

	m1 := r.get("worker-a")
	m2 := r.get("worker-b")

	assert.NotSame(t, m1, m2)
}

func TestHealthCheckpoint_NoopWithEmptySubsystem(t *testing.T) {
	ctx := context.Background()

	// Should not panic or create any monitor.
	HealthCheckpoint(ctx, nil)
	HealthCheckpoint(ctx, errors.New("test"))
}

func TestHealthCheckpoint_WithSubsystem(t *testing.T) {
	ctx := logutil.Start(context.Background(), "test-checkpoint")

	// Should not panic.
	HealthCheckpoint(ctx, nil)
	HealthCheckpoint(ctx, errors.New("test"))
}

func TestGetHealthMonitor_WithSubsystem(t *testing.T) {
	ctx := logutil.Start(context.Background(), "test-monitor")

	m := GetHealthMonitor(ctx)
	assert.NotNil(t, m)

	// Calling again should return the same instance.
	m2 := GetHealthMonitor(ctx)
	assert.Equal(t, m, m2)
}

func TestGetHealthMonitor_WithoutSubsystem(t *testing.T) {
	ctx := context.Background()

	m := GetHealthMonitor(ctx)

	// Returns a nil *healthMonitor which is still a valid HealthMonitor.
	// Checkpoint should be a no-op.
	m.Checkpoint(nil)
	m.Checkpoint(errors.New("test"))
}

func TestHealthMonitor_NilCheckpoint(t *testing.T) {
	var m *healthMonitor

	// Should not panic.
	m.Checkpoint(nil)
	m.Checkpoint(errors.New("test"))
}

func TestHealthMonitor_BackoffFromFiring(t *testing.T) {
	r := &healthRegistryImpl{
		monitors: map[string]*healthMonitor{},
	}
	m := r.get("test-backoff-firing")

	m.fire()
	assert.Equal(t, HealthStateFiring, m.state)

	m.Backoff()
	assert.Equal(t, HealthStateBackoff, m.state)
}

func TestHealthMonitor_BackoffNoopFromInit(t *testing.T) {
	r := &healthRegistryImpl{
		monitors: map[string]*healthMonitor{},
	}
	m := r.get("test-backoff-init")

	assert.Equal(t, HealthStateInit, m.state)

	m.Backoff()
	assert.Equal(t, HealthStateInit, m.state)
}

func TestHealthMonitor_BackoffNoopFromOK(t *testing.T) {
	r := &healthRegistryImpl{
		monitors: map[string]*healthMonitor{},
	}
	m := r.get("test-backoff-ok")

	m.resolve()
	assert.Equal(t, HealthStateOK, m.state)

	m.Backoff()
	assert.Equal(t, HealthStateOK, m.state)
}

func TestHealthMonitor_NilBackoff(t *testing.T) {
	var m *healthMonitor

	// Should not panic.
	m.Backoff()
}
