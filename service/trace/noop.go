package trace

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

var (
	NoopSpan trace.Span = trace.SpanFromContext(context.Background())
)

// NoopTracer is the implementation of the Tracer interface.
type NoopTracer struct{}

// NewNoopTracer creates a new tracer with no ops
func NewNoopTracer() *NoopTracer {
	return &NoopTracer{}
}

// Ready returns a channel that will close when the network stack is ready.
func (t *NoopTracer) Ready() <-chan struct{} {
	ready := make(chan struct{})
	close(ready)
	return ready
}

// Done returns a channel that will close when shutdown is complete.
func (t *NoopTracer) Done() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

func (t *NoopTracer) StartSpanFromContext(
	ctx context.Context,
	operationName SpanName,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return ctx, NoopSpan
}
