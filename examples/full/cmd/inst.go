package cmd

import (
	"context"
	"net/http"
	"strings"

	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/instutil"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/logutil"
)

// inst.go contains all functions for handling instrumentation (ie metrics and
// logs). All functions do not have an attached type and only rely on the
// context, so we do not have to pass another variable through the code. Having
// the instrumentation code in its own file gives a better separation between
// instrumentation code and actual buisniess logic. Each package can have its
// own ist.go file.

// We store the metric names in a constant, because this is our only reference
// in the code and using literal strings might be prone to errors.
const (
	instRequestsAcceptEncodingMetric = "request_accept_encoding_total"
	instRequestsMetric               = "request_total"
)

func InstInit(ctx context.Context) context.Context {
	ctx = instutil.NewCounterVec(ctx, instRequestsAcceptEncodingMetric, "encoding")
	ctx = instutil.NewCounter(ctx, instRequestsMetric)
	return ctx
}

func InstIndexRequest(ctx context.Context, r *http.Request) {
	logutil.Get(ctx).
		WithFields(logutil.FromStruct(r.Header)).
		Infof("got request")

	c, ok := instutil.Counter(ctx, instRequestsMetric)
	if ok {
		// We need to check whether the metric was initialized to avoid a nil
		// dereference.
		c.Inc()
	}

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

			cv, ok := instutil.CounterVec(ctx, instRequestsAcceptEncodingMetric)
			if ok {
				cv.WithLabelValues(accept).Inc()
			}
		}
	}
}
