package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/pkg/app/templates"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/webutil"
)

// UsersHandler handles the users pages
type UsersHandler struct {
	viewer *templates.Viewer
}

// NewUsersHandler creates a new users handler
func NewUsersHandler(
	viewer *templates.Viewer,
) *UsersHandler {
	return &UsersHandler{
		viewer: viewer,
	}
}

// Register registers the handler's routes
func (h *UsersHandler) Register(r chi.Router) {
	r.Get("/users", webutil.WrapView(h.handleUsersList))
}

// handleUsersList renders the users list page
func (h *UsersHandler) handleUsersList(r *http.Request) webutil.Response {
	// In a real app, we'd fetch users from a database
	users := []templates.User{
		{
			ID:        "1",
			Name:      "Alice Smith",
			Email:     "alice@example.com",
			CreatedAt: time.Now().Add(-72 * time.Hour),
		},
		{
			ID:        "2",
			Name:      "Bob Johnson",
			Email:     "bob@example.com",
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
		{
			ID:        "3",
			Name:      "Carol Williams",
			Email:     "carol@example.com",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
	}

	usersData := templates.UsersData{
		Users: users,
	}

	return templates.View(http.StatusOK, h.viewer.WithRequest(r).UsersPage(usersData))
}
