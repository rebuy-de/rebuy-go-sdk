package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/pkg/app/templates"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/webutil"
)

// IndexHandler handles the home page
type IndexHandler struct {
	viewer *templates.Viewer
}

// NewIndexHandler creates a new index handler
func NewIndexHandler(
	viewer *templates.Viewer,
) *IndexHandler {
	return &IndexHandler{
		viewer: viewer,
	}
}

// Register registers the handler's routes
func (h *IndexHandler) Register(r chi.Router) {
	r.Get("/", webutil.WrapView(h.handleIndex))
}

// handleIndex renders the home page
func (h *IndexHandler) handleIndex(r *http.Request) webutil.Response {
	return templates.View(http.StatusOK, h.viewer.WithRequest(r).HomePage())
}
