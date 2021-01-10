package export

import (
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/label"
)

// Empty default jaeger endpoint.
const defaultJaegerEndpoint = ""

// TracerShutdown is returned by exporter setup functions. This is called to
// shutdown the exporter.
type TracerShutdown func()

// InstallJaegerExporter installs opentelemetry exporter for Jaeger with the
// given service name. The returned TracerShutdown can be called to perform a
// flush of the exporter.
// This sets up a no-op provider by default. Set JAEGER_DISABLED=false
// and JAEGER_ENDPOINT=http://<service-address>:14268/api/traces environment
// variable to enable it a functional tracer provider.
func InstallJaegerExporter(serviceName string, opts ...jaeger.Option) (TracerShutdown, error) {
	// Default options.
	jOpts := []jaeger.Option{
		jaeger.WithProcess(jaeger.Process{
			ServiceName: serviceName,
			Tags: []label.KeyValue{
				label.String("exporter", "jaeger"),
			},
		}),
		// Disabled by default. Set env var JAEGER_DISABLED=false to enable it.
		jaeger.WithDisabled(true),
	}
	jOpts = append(jOpts, opts...)

	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint(defaultJaegerEndpoint),
		jOpts...,
	)
	if err != nil {
		return nil, err
	}

	return flush, nil
}
