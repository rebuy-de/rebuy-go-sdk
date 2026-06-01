package instutil

import (
	"net/http"

	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	ddotel "github.com/DataDog/dd-trace-go/v2/ddtrace/opentelemetry"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/cmdutil"
)

// Deprecated: InitDefaultTracer calls tracer.Start, which conflicts with
// ddotel.NewTracerProvider (it also calls tracer.Start and will discard this
// config, dropping all spans). Use InitOtelTracer instead, which builds the
// provider with these options and returns it for shutdown and dig registration.
func InitDefaultTracer() {
	tracer.Start(
		tracer.WithEnv("production"),
		tracer.WithService(cmdutil.Name),
		tracer.WithUDS("/var/run/datadog/apm.socket"),
	)

	InitHTTPTracing()
}

// InitOtelTracer builds a DataDog OpenTelemetry tracer provider with the
// standard rebuy options and enables outbound HTTP client tracing. Build it
// once in the production runner, shut it down on exit, and register it in dig
// so riverutil.NewRiverClient picks it up:
//
//	provider := instutil.InitOtelTracer()
//	defer func() { _ = provider.Shutdown() }()
//	digutil.ProvideValue[*ddotel.TracerProvider](c, provider)
func InitOtelTracer() *ddotel.TracerProvider {
	provider := ddotel.NewTracerProvider(
		tracer.WithEnv("production"),
		tracer.WithService(cmdutil.Name),
		tracer.WithUDS("/var/run/datadog/apm.socket"),
	)

	InitHTTPTracing()

	return provider
}

func InitHTTPTracing() {
	// This is a global action, since we are using the default client.
	_ = httptrace.WrapClient(http.DefaultClient,
		httptrace.WithResourceNamer(func(r *http.Request) string {
			return r.Host
		}))
}
