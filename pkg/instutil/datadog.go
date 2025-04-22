package instutil

import (
	"net/http"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func InitDefaultTracer() {
	tracer.Start(
		tracer.WithEnv("production"),
		tracer.WithService(cmdutil.Name),
		tracer.WithUDS("/var/run/datadog/apm.socket"),
	)

	InitHTTPTracing()
}

func InitHTTPTracing() {
	// This is a global action, since we are using the default client.
	_ = httptrace.WrapClient(http.DefaultClient,
		httptrace.RTWithResourceNamer(func(r *http.Request) string {
			return r.Host
		}))
}
