package tracing

import (
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Trace event names.
	infoEventName  = "info"
	errorEventName = "error"

	// Trace event attribute keys.
	messageKey   = "message"
	eventTypeKey = "event.type"
	nonStringKey = "non-string"

	// Attribute values.
	logEventTypeValue = "log" // Value for trace event type log.
)

// TracingLogger is a logger with tracing support. It captures all the logs and
// adds them into a tracing span.
type TracingLogger struct {
	logr.Logger
	trace.Span
}

// NewLogger creates and returns a TracingLogger.
func NewLogger(logger logr.Logger, span trace.Span) *TracingLogger {
	// Add tracing info in the logger if tracing is active.
	if span.IsRecording() {
		logger = logger.WithValues("SpanID", span.SpanContext().SpanID(), "TraceID", span.SpanContext().TraceID())
	}
	return &TracingLogger{
		Logger: logger,
		Span:   span,
	}
}

// Info implements the Logger interface.
func (t TracingLogger) Info(msg string, keysAndValues ...interface{}) {
	t.Logger.Info(msg, keysAndValues...)
	kvs := append(
		[]attribute.KeyValue{
			attribute.String(messageKey, msg),
			attribute.String(eventTypeKey, logEventTypeValue), // This helps identify an event as a log.
		},
		keyValues(keysAndValues...)...)
	t.Span.AddEvent(infoEventName, trace.WithAttributes(kvs...))
}

// Error implements the Logger interface.
func (t TracingLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	t.Logger.Error(err, msg, keysAndValues...)
	kvs := append(
		[]attribute.KeyValue{
			attribute.String(messageKey, msg),
			attribute.String(eventTypeKey, logEventTypeValue), // This helps identify an event as a log.
		},
		keyValues(keysAndValues...)...)
	t.Span.AddEvent(errorEventName, trace.WithAttributes(kvs...))
	t.Span.RecordError(err)
	t.Span.SetStatus(codes.Error, err.Error())
}

// V implements the Logger interface.
func (t TracingLogger) V(level int) logr.Logger {
	return TracingLogger{Logger: t.Logger.V(level), Span: t.Span}
}

// WithValues implements the Logger interface.
func (t TracingLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	t.Span.SetAttributes(keyValues(keysAndValues...)...)
	return TracingLogger{Logger: t.Logger.WithValues(keysAndValues...), Span: t.Span}
}

// WithName implements the Logger interface.
func (t TracingLogger) WithName(name string) logr.Logger {
	t.Span.SetAttributes(attribute.String("name", name))
	return TracingLogger{Logger: t.Logger.WithName(name), Span: t.Span}
}

// keyValues converts the keysAndValues input from logger into a slice of
// opentelemetry attributes.
func keyValues(keysAndValues ...interface{}) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			// The key isn't a string. Unexpected value type.
			key = nonStringKey
		}
		attrs = append(attrs, attribute.Any(key, keysAndValues[i+1]))
	}
	return attrs
}
