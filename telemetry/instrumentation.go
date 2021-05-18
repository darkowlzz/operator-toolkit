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

// Instrumentation provides instrumentation builder consisting of tracer, meter
// and logger.
type Instrumentation struct {
	trace  trace.Tracer
	metric metric.Meter
	log    logr.Logger
}

// NewInstrumentation constructs and returns a new Instrumentation. The tracer
// and meter can be configured by passing trace or meter providers.
func NewInstrumentation(name string, tp trace.TracerProvider, mp metric.MeterProvider, log logr.Logger) *Instrumentation {
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
		log:    log.WithValues("library", name),
	}
}

// Start creates and returns a span, a meter and a tracing logger.
func (i *Instrumentation) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span, metric.Meter, logr.Logger) {
	ctx, span := i.trace.Start(ctx, name, opts...)
	// Use the created span to create a tracing logger with the span name.
	tl := tracing.NewLogger(i.log.WithName(name), span)
	return ctx, span, i.metric, tl
}
