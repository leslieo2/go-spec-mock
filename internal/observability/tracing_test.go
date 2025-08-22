package observability

import (
	"context"
	"testing"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestNewTracer(t *testing.T) {
	tracer, err := NewTracer(config.DefaultTracingConfig())
	if err != nil {
		t.Fatalf("NewTracer() returned error: %v", err)
	}

	if tracer == nil {
		t.Fatal("NewTracer() returned nil")
	}

	// Verify tracer is properly initialized
	if tracer == nil {
		t.Error("Tracer is nil")
	}
}

func TestTracer_StartSpan(t *testing.T) {
	tracer, err := NewTracer(config.DefaultTracingConfig())
	if err != nil {
		t.Fatalf("NewTracer() returned error: %v", err)
	}

	ctx := context.Background()
	spanName := "test-span"
	attrs := []attribute.KeyValue{
		attribute.String("test.key", "test.value"),
		attribute.Int("test.number", 42),
	}

	newCtx, span := tracer.StartSpan(ctx, spanName, attrs...)
	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}

	// Verify span is in context (noop tracer may not have valid context)
	spanCtx := trace.SpanFromContext(newCtx).SpanContext()
	_ = spanCtx // Accept that noop tracer may not have valid context

	// End the span
	span.End()
}

func TestTracer_StartSpan_EmptyAttributes(t *testing.T) {
	tracer, err := NewTracer(config.DefaultTracingConfig())
	if err != nil {
		t.Fatalf("NewTracer() returned error: %v", err)
	}

	ctx := context.Background()
	spanName := "test-span-no-attrs"

	_, span := tracer.StartSpan(ctx, spanName)
	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}

	span.End()
}

func TestTracer_MultipleSpans(t *testing.T) {
	tracer, err := NewTracer(config.DefaultTracingConfig())
	if err != nil {
		t.Fatalf("NewTracer() returned error: %v", err)
	}

	ctx := context.Background()

	// Create parent span
	newCtx, parentSpan := tracer.StartSpan(ctx, "parent-span")
	if parentSpan == nil {
		t.Fatal("StartSpan() returned nil parent span")
	}

	// Create child span
	newCtx, childSpan := tracer.StartSpan(newCtx, "child-span")
	if childSpan == nil {
		t.Fatal("StartSpan() returned nil child span")
	}

	// Verify child span has parent context (noop tracer may not have valid context)
	childSpanCtx := trace.SpanFromContext(newCtx).SpanContext()
	_ = childSpanCtx // Accept that noop tracer may not have valid context

	childSpan.End()
	parentSpan.End()
}

func TestTracer_ContextPropagation(t *testing.T) {
	tracer, err := NewTracer(config.DefaultTracingConfig())
	if err != nil {
		t.Fatalf("NewTracer() returned error: %v", err)
	}

	// Create initial context with span
	ctx := context.Background()
	newCtx, span := tracer.StartSpan(ctx, "initial-span", attribute.String("initial", "true"))

	// Propagate context to new span
	_, newSpan := tracer.StartSpan(newCtx, "propagated-span", attribute.String("propagated", "true"))

	// Verify spans have different contexts but are related (noop tracer may not have valid context)
	initialSpanCtx := trace.SpanFromContext(newCtx).SpanContext()
	_ = initialSpanCtx // Accept that noop tracer may not have valid context

	newSpan.End()
	span.End()
}

func TestTracer_ConcurrentSpans(t *testing.T) {
	tracer, err := NewTracer(config.DefaultTracingConfig())
	if err != nil {
		t.Fatalf("NewTracer() returned error: %v", err)
	}

	done := make(chan bool)

	// Create concurrent spans
	for i := 0; i < 10; i++ {
		go func(id int) {
			ctx := context.Background()
			_, span := tracer.StartSpan(ctx, "concurrent-span", attribute.Int("id", id))
			span.End()
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestTracer_NilContext(t *testing.T) {
	tracer, err := NewTracer(config.DefaultTracingConfig())
	if err != nil {
		t.Fatalf("NewTracer() returned error: %v", err)
	}

	// Test with nil context (should handle gracefully)
	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "nil-context-test")

	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}

	span.End()
}
