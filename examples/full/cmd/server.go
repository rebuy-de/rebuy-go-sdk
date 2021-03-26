package cmd

import (
	"context"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/redisutil"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/webutil"
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
	// running routines and should used once on program startup. We have to use
	// ctxRoot, because this is what should canceled first, if any error
	// occours.
	group, ctxRoot := errgroup.WithContext(ctxRoot)

	// Set up the admin API and use the root context to make sure it gets terminated first.
	webutil.AdminAPIListenAndServe(ctxRoot, group, cancel)

	// Other background processes use the main context.
	s.setupHTTPServer(ctx, group)

	return errors.WithStack(group.Wait())
}

func (s *Server) setupHTTPServer(ctx context.Context, group *errgroup.Group) {
	// It is a good practice to init a new context logger for a new background
	// process, so we can see what triggered a specific log message later.
	ctx = logutil.Start(ctx, "http-server")

	router := httprouter.New()
	router.GET("/", s.handleIndex)
	router.ServeFiles("/assets/*filepath", http.FS(s.AssetFS))

	group.Go(func() error {
		logutil.Get(ctx).Info("http server listening on port 8080")
		return errors.WithStack(webutil.ListenAndServerWithContext(
			ctx, "0.0.0.0:8080", router))
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	InstIndexRequest(r.Context(), r)

	t, err := template.ParseFS(s.TemplateFS, "index.html")
	if webutil.RespondError(w, err) {
		return
	}

	err = t.Execute(w, nil)
	if webutil.RespondError(w, err) {
		return
	}
}
