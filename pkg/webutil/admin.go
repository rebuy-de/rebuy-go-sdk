package webutil

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/logutil"
)

func AdminAPIListenAndServe(ctx context.Context, healthy ...func() error) func() {
	ctx = logutil.Start(ctx, "admin-api")
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if ctx.Err() != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "SHUTTING DOWN")
			return
		}

		for _, h := range healthy {
			err := h()
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintln(w, err.Error())
				return
			}
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

	// The admin api gets a its own context, because we want to delay the
	// server shutdown as long as possible. The reason for this is that Istio
	// starts to block all outgoing connections as soon as there is no
	// listening server anymore.
	bg, cancel := context.WithCancel(context.Background())

	go func() {
		logutil.Get(ctx).Debugf("admin api listening on port 8090")

		err := ListenAndServeWithContext(bg, "0.0.0.0:8090", mux)
		if err != nil {
			logutil.Get(ctx).Error(err.Error())
		}
	}()

	return cancel
}
