package telemetry

import (
	"context"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/trace"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/darkowlzz/operator-toolkit/telemetry/tracing"
)

// Name of the logger library key.
const logLibraryKey = "library"

// Instrumentation provides instrumentation builder consisting of tracer, meter
// and logger.
type Instrumentation struct {
	trace  trace.Tracer
	metric metric.Meter
	log    logr.Logger
}

// NewInstrumentation constructs and returns a new Instrumentation based on the
// given providers.
func NewInstrumentationWithProviders(name string, tp trace.TracerProvider, mp metric.MeterProvider, log logr.Logger) *Instrumentation {
	if tp == nil {
		tp = otel.GetTracerProvider()
	}
	if mp == nil {
		mp = global.GetMeterProvider()
	}
	if log == nil {
		log = ctrl.Log
	}
	return &Instrumentation{
		trace:  tp.Tracer(name),
		metric: mp.Meter(name),
		log:    log.WithValues(logLibraryKey, name),
	}
}

// NewInstrumentation constructs and returns a new Instrumentation with default
// providers.
func NewInstrumentation(name string) *Instrumentation {
	return &Instrumentation{
		trace:  otel.GetTracerProvider().Tracer(name),
		metric: global.GetMeterProvider().Meter(name),
		log:    ctrl.Log.WithValues(logLibraryKey, name),
	}
}

// Start creates and returns a span, a meter and a tracing logger.
func (i *Instrumentation) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span, metric.Meter, logr.Logger) {
	ctx, span := i.trace.Start(ctx, name, opts...)
	// Use the created span to create a tracing logger with the span name.
	tl := tracing.NewLogger(i.log.WithValues("spanName", name), span)
	return ctx, span, i.metric, tl
}
