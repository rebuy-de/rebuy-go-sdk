package cmd

import (
	"context"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/logutil"
)

// inst.go contains all functions for handling instrumentation (ie metrics and
// logs). Having the instrumentation code in its own file gives a better
// separation between instrumentation code and actual buisniess logic. Each
// package can have its own inst.go file.

var (
	instRequestsAcceptEncodingMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "rebuy_go_sdk",
		Subsystem: "requests",
		Name:      "encoding_total",
	}, []string{"encoding"})
	instRequestsMetric = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "rebuy_go_sdk",
		Subsystem: "requests",
		Name:      "total",
	})
)

func InstIndexRequest(ctx context.Context, r *http.Request) {
	logutil.Get(ctx).
		WithFields(logutil.FromStruct(r.Header)).
		Infof("got request")

	instRequestsMetric.Inc()

	// We have a metric that lists all accepted encodings. Together with the
	// total request count we get a ratio of each encoding. This is a good
	// demonstration for instrumentation complexity that should not be part of
	// the buissiness logic.
	for _, acceptList := range r.Header.Values("Accept-Encoding") {
		for _, accept := range strings.Split(acceptList, ",") {
			accept = strings.TrimSpace(accept)
			if accept == "" {
				continue
			}

			instRequestsAcceptEncodingMetric.WithLabelValues(accept).Inc()
		}
	}
}
