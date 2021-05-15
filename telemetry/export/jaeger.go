package export

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

// TracerShutdown is returned by exporter setup functions. This is called to
// shutdown the exporter.
type TracerShutdown func()

// InstallJaegerExporter installs opentelemetry exporter for Jaeger with the
// given service name. The returned TracerShutdown can be called to perform a
// flush of the exporter.
// This sets up a no-op provider by default. Set DISABLE_TRACING=false
// and OTEL_EXPORTER_JAEGER_ENDPOINT=http://<service-address>:14268/api/traces
// environment variables to enable a functional tracer provider.
// For details about configuring jaeger using otel environment variables, refer
// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/sdk-environment-variables.md#jaeger-exporter
func InstallJaegerExporter(serviceName string, tpOpts ...sdktrace.TracerProviderOption) (TracerShutdown, error) {
	// If tracing is not enabled, skip setting up a Tracer Provider.
	if getEnvAsBool(envDisableTracing, true) {
		return func() {}, nil
	}

	exp, err := jaeger.NewRawExporter(jaeger.WithCollectorEndpoint())
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("exporter", "jaeger"),
		)),
	)

	// Register the TraceProvider as the global.
	otel.SetTracerProvider(tp)

	// Return shutdown function.
	return func() {
		// TODO: Make this timeout configurable.
		// New context, do not make the application hang when it is shutdown.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}, nil
}
