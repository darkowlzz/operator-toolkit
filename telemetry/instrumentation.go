package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Instrumentation provides instrumentation builder consisting of tracer and
// meter.
type Instrumentation struct {
	trace.Tracer
	metric.Meter
}

// NewInstrumentation constructs and returns a new Instrumentation. The tracer
// and meter can be configured by passing trace or meter providers.
func NewInstrumentation(name string, tp trace.TracerProvider, mp metric.MeterProvider) *Instrumentation {
	if tp == nil {
		tp = otel.GetTracerProvider()
	}
	if mp == nil {
		mp = otel.GetMeterProvider()
	}
	return &Instrumentation{
		Tracer: tp.Tracer(name),
		Meter:  mp.Meter(name),
	}
}
