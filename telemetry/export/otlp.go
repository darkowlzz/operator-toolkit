package export

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
)

// InstallOTLPExporter installs opentelemetry exporter for OTLP collector with
// the given service name. The returned TracerShutdown can be called to perform
// a flush of the exporter.
// TODO: Make it more configurable and document usage.
func InstallOTLPExporter(serviceName string, expOpts ...otlp.ExporterOption) (TracerShutdown, error) {
	ctx := context.Background()

	exp, err := otlp.NewExporter(ctx, expOpts...)
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
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	pusher := push.New(
		basic.New(
			simple.NewWithExactDistribution(),
			exp,
		),
		exp,
		push.WithPeriod(2*time.Second),
	)

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	pusher.Start()

	return func() {
		err := tracerProvider.Shutdown(ctx)
		if err != nil {
			log.Fatalf("failed to stop trace provider: %v", err)
		}

		pusher.Stop()
		err = exp.Shutdown(ctx)
		if err != nil {
			log.Fatalf("failed to stop trace exporter: %v", err)
		}
	}, nil
}
