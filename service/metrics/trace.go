package metrics

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"time"

	archiveTrace "github.com/onflow/flow-archive/service/trace"
)

// Tracer is a generic tracer implementation for the Archive API servers
type Tracer struct {
	tracer   trace.Tracer
	shutdown func(context.Context) error
	log      zerolog.Logger
}

func NewTracer(log zerolog.Logger, serviceName string) (*Tracer, error) {
	ctx := context.TODO()
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// OLTP trace gRPC client initialization. Connection parameters for the exporter are extracted
	// from environment variables. e.g.: `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`.
	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Debug().Err(err).Msg("tracing error")
	}))
	return &Tracer{
		tracer:   tracerProvider.Tracer(""),
		shutdown: tracerProvider.Shutdown,
		log:      log,
	}, nil
}

// Ready returns a channel that will close when the network stack is ready.
func (t *Tracer) Ready() <-chan struct{} {
	ready := make(chan struct{})
	close(ready)
	return ready
}

// Done returns a channel that will close when shutdown is complete.
func (t *Tracer) Done() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := t.shutdown(ctx); err != nil {
			t.log.Error().Err(err).Msg("failed to shutdown tracer")
		}
		close(done)
	}()
	return done
}

func (t *Tracer) StartSpanFromContext(
	ctx context.Context,
	operationName archiveTrace.SpanName,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, string(operationName), opts...)
}
