package instutil

import (
	"net/http"

	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/cmdutil"
)

// Deprecated: InitDefaultTracer calls tracer.Start, which conflicts with
// ddotel.NewTracerProvider (it also calls tracer.Start and will discard this
// config, dropping all spans). Build the provider with the tracer options
// instead:
//
//	provider := ddotel.NewTracerProvider(
//		tracer.WithEnv("production"),
//		tracer.WithService(cmdutil.Name),
//		tracer.WithUDS("/var/run/datadog/apm.socket"),
//	)
//	defer func() { _ = provider.Shutdown() }()
//
// For outbound HTTP client tracing, call InitHTTPTracing directly.
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
		httptrace.WithResourceNamer(func(r *http.Request) string {
			return r.Host
		}))
}
