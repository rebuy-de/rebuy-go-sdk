package webutil

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/logutil"
	"golang.org/x/sync/errgroup"
)

// ListenAndServerWithContext does the same as http.ListenAndServe with the
// difference that is properly utilises the context. This means it does a
// graceful shutdown when the context is done and a context cancellation gets
// propagated down to the actual request context.
func ListenAndServerWithContext(ctx context.Context, addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		BaseContext: func(_ net.Listener) context.Context {
			ctx := logutil.Start(ctx, "request")
			return ctx
		},
	}

	grp, ctx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		err := server.ListenAndServe()
		if err == http.ErrServerClosed {
			// We do not want to print an error on graceful shutdown.
			return nil
		}

		return errors.WithStack(err)
	})

	grp.Go(func() error {
		<-ctx.Done()

		logutil.Get(ctx).Warn("Got shutdown signal")
		time.Sleep(3 * time.Second) // Give systems some time to populate shutdown.

		logutil.Get(ctx).Debug("Shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return errors.WithStack(server.Shutdown(shutdownCtx))
	})

	return errors.Wrap(grp.Wait(), "http server failed")
}
