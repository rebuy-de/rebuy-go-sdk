package runutil

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
)

const (
	promNamespace       = "rebuy_go_sdk"
	promHealthSubsystem = "health"
)

const (
	HealthStateInit   = "init"
	HealthStateOK     = "ok"
	HealthStateFiring = "firing"
)

var (
	instHealthCheckpointsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promHealthSubsystem,
		Name:      "checkpoints_total",
	}, []string{"name", "state"})

	instHealthState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: promNamespace,
		Subsystem: promHealthSubsystem,
		Name:      "state",
	}, []string{"name", "state"})
)

var healthRegistry = &healthRegistryImpl{
	monitors: map[string]*healthMonitor{},
}

// HealthMonitor tracks the health state of a worker via Prometheus metrics.
type HealthMonitor interface {
	Checkpoint(err error)
}

// HealthCheckpoint records a health checkpoint for the current subsystem.
// It is a no-op if the context has no subsystem set.
func HealthCheckpoint(ctx context.Context, err error) {
	name := logutil.GetSubsystem(ctx)
	if name == "" {
		return
	}

	healthRegistry.get(name).Checkpoint(err)
}

// GetHealthMonitor returns the health monitor for the current subsystem.
// Useful for long-running workers that need to pass a monitor to helpers.
func GetHealthMonitor(ctx context.Context) HealthMonitor {
	name := logutil.GetSubsystem(ctx)
	if name == "" {
		return (*healthMonitor)(nil)
	}

	return healthRegistry.get(name)
}

type healthRegistryImpl struct {
	monitors map[string]*healthMonitor
	mu       sync.Mutex
}

func (r *healthRegistryImpl) get(name string) *healthMonitor {
	r.mu.Lock()
	defer r.mu.Unlock()

	monitor, ok := r.monitors[name]
	if !ok {
		monitor = newHealthMonitor(name)
		r.monitors[name] = monitor
	}

	return monitor
}

func newHealthMonitor(name string) *healthMonitor {
	m := &healthMonitor{
		name: name,
	}

	for _, state := range []string{HealthStateInit, HealthStateOK, HealthStateFiring} {
		// Register zero values immediately to avoid null values in Prometheus.
		instHealthCheckpointsTotal.
			WithLabelValues(name, state).
			Add(0)
		instHealthState.
			WithLabelValues(name, state).
			Set(0)
	}

	instHealthState.
		WithLabelValues(name, HealthStateInit).
		Set(1)

	return m
}

type healthMonitor struct {
	name string
}

func (m *healthMonitor) Checkpoint(err error) {
	if m == nil {
		return
	}

	if err == nil {
		m.resolve()
	} else {
		m.fire()
	}
}

func (m *healthMonitor) resolve() {
	instHealthCheckpointsTotal.
		WithLabelValues(m.name, HealthStateOK).
		Add(1)
	instHealthState.
		WithLabelValues(m.name, HealthStateInit).
		Set(0)
	instHealthState.
		WithLabelValues(m.name, HealthStateOK).
		Set(1)
	instHealthState.
		WithLabelValues(m.name, HealthStateFiring).
		Set(0)
}

func (m *healthMonitor) fire() {
	instHealthCheckpointsTotal.
		WithLabelValues(m.name, HealthStateFiring).
		Add(1)
	instHealthState.
		WithLabelValues(m.name, HealthStateInit).
		Set(0)
	instHealthState.
		WithLabelValues(m.name, HealthStateOK).
		Set(0)
	instHealthState.
		WithLabelValues(m.name, HealthStateFiring).
		Set(1)
}
