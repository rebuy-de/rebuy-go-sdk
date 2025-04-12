package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rebuy-de/rebuy-go-sdk/v8/examples/full/pkg/app/templates"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/webutil"
)

// HealthHandler handles the health status page
type HealthHandler struct {
	viewer *templates.Viewer
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(
	viewer *templates.Viewer,
) *HealthHandler {
	return &HealthHandler{
		viewer: viewer,
	}
}

// Register registers the handler's routes
func (h *HealthHandler) Register(r chi.Router) {
	r.Get("/health", webutil.WrapView(h.handleHealth))
}

// handleHealth renders the health status page
func (h *HealthHandler) handleHealth(r *http.Request) webutil.Response {
	// In a real app, we'd check actual components and workers
	components := []templates.Component{
		{
			Name:      "Database",
			Healthy:   true,
			LastCheck: time.Now().Add(-5 * time.Minute),
		},
		{
			Name:      "Redis",
			Healthy:   true,
			LastCheck: time.Now().Add(-3 * time.Minute),
		},
		{
			Name:      "External API",
			Healthy:   false,
			LastCheck: time.Now().Add(-10 * time.Minute),
		},
	}

	workers := []templates.Worker{
		{
			Name:    "Data Sync Worker",
			Running: true,
			LastRun: time.Now().Add(-15 * time.Minute),
		},
		{
			Name:    "Periodic Task Worker",
			Running: true,
			LastRun: time.Now().Add(-5 * time.Minute),
		},
	}

	healthData := templates.HealthData{
		Components: components,
		Workers:    workers,
	}

	return templates.View(http.StatusOK, h.viewer.WithRequest(r).HealthPage(healthData))
}
