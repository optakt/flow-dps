package trace

import (
	"context"
	otelTrace "go.opentelemetry.io/otel/trace"
)

type SpanName string

// Tracer interface for tracers in the Archive Node
type Tracer interface {
	// Ready commences startup
	Ready() <-chan struct{}

	// Done commences shutdown
	Done() <-chan struct{}

	StartSpanFromContext(
		ctx context.Context,
		operationName SpanName,
		opts ...otelTrace.SpanStartOption,
	) (context.Context, otelTrace.Span)
}
