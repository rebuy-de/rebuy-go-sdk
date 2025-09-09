package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/pkg/app/templates"
	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/pkg/dal/sqlc"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/webutil"
)

// UsersHandler handles the users pages
type UsersHandler struct {
	viewer  *templates.Viewer
	queries *sqlc.Queries
}

// NewUsersHandler creates a new users handler
func NewUsersHandler(
	viewer *templates.Viewer,
	queries *sqlc.Queries,
) *UsersHandler {
	return &UsersHandler{
		viewer:  viewer,
		queries: queries,
	}
}

// Register registers the handler's routes
func (h *UsersHandler) Register(r chi.Router) {
	r.Get("/users", webutil.WrapView(h.handleUsersList))
}

// handleUsersList renders the users list page
func (h *UsersHandler) handleUsersList(r *http.Request) webutil.Response {
	ctx := r.Context()
	logger := logutil.Get(ctx)

	// Fetch users from database
	dbUsers, err := h.queries.ListUsers(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to fetch users from database")
		return webutil.ViewError(http.StatusInternalServerError, err)
	}

	// Convert database users to template users
	users := make([]templates.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = templates.User{
			ID:        dbUser.ID.String(),
			Name:      dbUser.Name,
			Email:     dbUser.Email,
			CreatedAt: dbUser.CreatedAt,
		}
	}

	usersData := templates.UsersData{
		Users: users,
	}

	return templates.View(http.StatusOK, h.viewer.WithRequest(r).UsersPage(usersData))
}
