package webutil

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/logutil"
	"golang.org/x/sync/errgroup"
)

func AdminAPIListenAndServe(ctx context.Context, group *errgroup.Group, fnDone func()) {
	ctx = logutil.Start(ctx, "admin-api")
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if ctx.Err() != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "SHUTTING DOWN")
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Copied from init in https://golang.org/src/net/http/pprof/pprof.go,
	// because the package does not allow specifying a mux.
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	group.Go(func() error {
		defer fnDone()

		logutil.Get(ctx).Debugf("admin api listening on port 8090")

		return errors.WithStack(ListenAndServerWithContext(
			ctx, "0.0.0.0:8090", mux))
	})
}
