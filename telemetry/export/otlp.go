package export

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

// InstallOTLPExporter installs opentelemetry exporter for OTLP collector with
// the given service name. The returned TracerShutdown can be called to perform
// a flush of the exporter.
// TODO: Make it more configurable and document usage.
func InstallOTLPExporter(serviceName string, driverOpts ...otlpgrpc.Option) (TracerShutdown, error) {
	// If tracing is not enabled, skip setting up a Tracer Provider.
	if getEnvAsBool(envDisableTracing, true) {
		return func() {}, nil
	}

	ctx := context.Background()

	driver := otlpgrpc.NewDriver(driverOpts...)

	exp, err := otlp.NewExporter(ctx, driver)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	cont := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exp,
		),
		controller.WithExporter(exp),
		controller.WithCollectPeriod(2*time.Second),
	)

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	if err := cont.Start(ctx); err != nil {
		return nil, err
	}

	return func() {
		if err := cont.Stop(ctx); err != nil {
			log.Fatalf("failed to stop controller: %v", err)
		}

		if err = tracerProvider.Shutdown(ctx); err != nil {
			log.Fatalf("failed to stop TraceProvider: %v", err)
		}
	}, nil
}
