package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/redisutil"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/webutil"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	RedisClient *redis.Client
	RedisPrefix redisutil.Prefix

	AssetFS    fs.FS
	TemplateFS fs.FS
}

func (s *Server) Run(ctxRoot context.Context) error {
	// Creating a new context, so we can have two stages for the graceful
	// shutdown. First is to make pod unready (within the admin api) and the
	// seconds is all the rest.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = InstInit(ctx)

	// Using a errors group is a good practice to manage multiple parallel
	// running routines and should used once on program startup.
	group, ctx := errgroup.WithContext(ctx)

	// Set up the admin API and use the root context to make sure it gets
	// terminated first.  We have to use ctxRoot, because this is what should
	// canceled first, if any error occours. Afterwards it uses cancel() to
	// cancel ctx context.
	webutil.AdminAPIListenAndServe(ctxRoot, group, cancel)

	// Other background processes use the main context.
	s.setupHTTPServer(ctx, group)

	return errors.WithStack(group.Wait())
}

func (s *Server) setupHTTPServer(ctx context.Context, group *errgroup.Group) {
	// It is a good practice to init a new context logger for a new background
	// process, so we can see what triggered a specific log message later.
	ctx = logutil.Start(ctx, "http-server")

	// Prepare some interfaces to later use.
	vh := webutil.NewViewHandler(s.TemplateFS,
		webutil.SimpleTemplateFuncMap("prettyTime", PrettyTimeTemplateFunction),
	)

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Get("/", vh.Wrap(s.handleIndex))
	router.Get("/json", vh.Wrap(s.handleJSON))
	router.Get("/redirect", vh.Wrap(s.handleRedirect))
	router.Get("/error", vh.Wrap(s.handleError))
	router.Mount("/assets", http.StripPrefix("/assets", http.FileServer(http.FS(s.AssetFS))))

	group.Go(func() error {
		logutil.Get(ctx).Info("http server listening on port 8080")
		return errors.WithStack(webutil.ListenAndServerWithContext(
			ctx, "0.0.0.0:8080", router))
	})
}

func (s *Server) timeModel() any {
	return map[string]interface{}{
		"now": time.Now(),
	}
}

func (s *Server) handleIndex(v *webutil.View, r *http.Request) webutil.Response {
	return v.HTML(http.StatusOK, "index.html", s.timeModel())
}

func (s *Server) handleJSON(v *webutil.View, r *http.Request) webutil.Response {
	return v.JSON(http.StatusOK, s.timeModel())
}

func (s *Server) handleRedirect(v *webutil.View, r *http.Request) webutil.Response {
	return v.Redirect(http.StatusTemporaryRedirect, "/")
}

func (s *Server) handleError(v *webutil.View, r *http.Request) webutil.Response {
	return v.Error(http.StatusBadRequest, fmt.Errorf("oh no"))
}
