package webutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"

	// instutil import ensures the init function of that package is run, which adds the toolstack metrics
	_ "github.com/rebuy-de/rebuy-go-sdk/v9/pkg/instutil"
)

type adminAPIListenAndServeOptions struct {
	host string
	port string
}

type AdminAPIListenAndServeOption func(*adminAPIListenAndServeOptions)

func WithPort(port string) AdminAPIListenAndServeOption {
	return func(o *adminAPIListenAndServeOptions) {
		o.port = port
	}
}

func WithHost(host string) AdminAPIListenAndServeOption {
	return func(o *adminAPIListenAndServeOptions) {
		o.host = host
	}
}

func AdminAPIListenAndServe(ctx context.Context, opts ...AdminAPIListenAndServeOption) {
	config := adminAPIListenAndServeOptions{
		host: "0.0.0.0",
		port: "8090",
	}

	for _, o := range opts {
		o(&config)
	}

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

	// The admin api gets a its own context, because we want to delay the
	// server shutdown as long as possible. The reason for this is that Istio
	// starts to block all outgoing connections as soon as there is no
	// listening server anymore. Also a graceful shutdown is not needed for the
	// admin API, so it is also not necessary to cancel the context.
	bg := context.Background()

	go func() {
		serverAddress := net.JoinHostPort(config.host, config.port)

		logutil.Get(ctx).Debug("admin api listening", "address", serverAddress)

		err := ListenAndServeWithContext(bg, serverAddress, mux)
		if err != nil {
			logutil.Get(ctx).Error(err.Error())
		}
	}()
}
