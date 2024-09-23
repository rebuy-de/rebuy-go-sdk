package webutil

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/runutil"
	"go.uber.org/dig"
	"golang.org/x/sync/errgroup"
	chitrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-chi/chi.v5"
)

// ListenAndServeWithContext does the same as http.ListenAndServe with the
// difference that is properly utilises the context. This means it does a
// graceful shutdown when the context is done and a context cancellation gets
// propagated down to the actual request context.
func ListenAndServeWithContext(ctx context.Context, addr string, handler http.Handler) error {
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

// AssetFS is the fs.FS that is used to server assets. It is a separate type to support dependency injection.
type AssetFS fs.FS

// AssetPathPrefix is the prefix that is prepended to each asset path. It is suggested to use the commit hash in
// production and "dev" for development. This way the server always serves new assets on rollout, even if they are still
// cached.
type AssetPathPrefix string

// AssetCacheDuration defines the duration for the caching headers of assets. It is suggested to use a long time (e.g. a
// year) for production and a second for development.
type AssetCacheDuration time.Duration

// AssetDefaultProd provides the suggested defaults for production environments.
func AssetDefaultProd() (AssetPathPrefix, AssetCacheDuration) {
	return AssetPathPrefix(cmdutil.CommitHash),
		AssetCacheDuration(365 * 24 * time.Hour)
}

// AssetDefaultDev provides the suggested defaults for development environments.
func AssetDefaultDev() (AssetPathPrefix, AssetCacheDuration) {
	return AssetPathPrefix("dev"),
		AssetCacheDuration(time.Second)
}

// Middlewares defines all chi middlewares. It needs to be provided as list to dig and not as single middlewares, since
// order matters and dig does not guarantee any ordering. To just append middlewares, you need to append them to the
// ones from `DefaultMiddlewares`
type Middlewares []func(http.Handler) http.Handler

func DefaultMiddlewares() Middlewares {
	return Middlewares{
		middleware.Compress(7),
		chitrace.Middleware(),

		// HX-Target is set by HTMX and used by us to decide whether to send the
		// whole page or just a frame.
		middleware.SetHeader("vary", "hx-target"),
	}
}

// Server is a web server targeted on projects that have a user-facing web interface. It supports dependency injection
// using dig.
type Server struct {
	AssetFS            AssetFS
	AssetPathPrefix    AssetPathPrefix
	AssetCacheDuration AssetCacheDuration
	Handlers           []Handler
	Middlewares        Middlewares
}

// ServerParams defines all parameters that are needed for the Server. Its fields can be injected using dig.
type ServerParams struct {
	dig.In

	AssetFS            AssetFS
	AssetPathPrefix    AssetPathPrefix
	AssetCacheDuration AssetCacheDuration
	Handlers           []Handler   `group:"handler"`
	Middlewares        Middlewares `optional:"true"`
}

// Handler is the interface that HTTP handlers need to implement to get picked up and served by the Server.
type Handler interface {
	Register(chi.Router)
}

// Helper to provide a handler to dependency injection.
func ProvideHandler(c *dig.Container, fn any) error {
	return c.Provide(fn, dig.Group("handler"), dig.As(new(Handler)))
}

func NewServer(p ServerParams) *Server {
	middlewares := p.Middlewares
	if len(middlewares) == 0 {
		middlewares = DefaultMiddlewares()
	}

	return &Server{
		AssetFS:            p.AssetFS,
		AssetPathPrefix:    p.AssetPathPrefix,
		AssetCacheDuration: p.AssetCacheDuration,
		Handlers:           p.Handlers,
		Middlewares:        middlewares,
	}
}

// Workers defines the workers, making it compatible with runutil.
func (s *Server) Workers() []runutil.Worker {
	return []runutil.Worker{s}
}

func (s *Server) Run(ctx context.Context) error {
	AdminAPIListenAndServe(ctx)

	// Delay the context cancel by 5s to give Kubernetes some time to redirect
	// traffic to another pod.
	ctx = cmdutil.ContextWithDelay(ctx, 5*time.Second)

	router := chi.NewRouter()
	for _, mw := range s.Middlewares {
		router.Use(mw)
	}

	for _, h := range s.Handlers {
		h.Register(router)
	}

	assetPath := "/assets/" + string(s.AssetPathPrefix)
	cacheControl := fmt.Sprintf("public, max-age=%d",
		time.Duration(s.AssetCacheDuration).Truncate(time.Second)/time.Second)
	router.Route(assetPath, func(router chi.Router) {
		router.Use(middleware.SetHeader("Cache-Control", cacheControl))
		router.Handle("/*", http.StripPrefix(assetPath, http.FileServer(http.FS(s.AssetFS))))
	})

	logutil.Get(ctx).Info("http server listening on port 8080")
	return errors.WithStack(ListenAndServeWithContext(
		ctx, "0.0.0.0:8080", router))
}
