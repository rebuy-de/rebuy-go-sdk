package riverutil

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"riverqueue.com/riverui"
)

func createRiverUIHandler(client *river.Client[pgx.Tx]) (*riverui.Handler, error) {
	endpoints := riverui.NewEndpoints(client, nil)
	opts := &riverui.HandlerOpts{
		Endpoints: endpoints,
		DevMode:   false,
		Prefix:    "/riverui",
		Logger:    slog.New(slog.DiscardHandler),
	}
	return riverui.NewHandler(opts)
}

type Handler struct {
	riverUIHandler *riverui.Handler
}

func NewHandler(ctx context.Context, client *river.Client[pgx.Tx]) (*Handler, error) {
	handler, err := createRiverUIHandler(client)
	if err != nil {
		return nil, err
	}

	err = handler.Start(ctx)
	if err != nil {
		return nil, err
	}

	return &Handler{
		riverUIHandler: handler,
	}, nil
}

func (h *Handler) Register(router chi.Router) {
	router.Mount("/riverui/", h.riverUIHandler)
}
