package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
oteltrace "go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	tracer oteltrace.Tracer
}

type TraceConfig struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	Exporter   string `json:"exporter" yaml:"exporter"`
	ServiceName string `json:"service_name" yaml:"service_name"`
	Environment string `json:"environment" yaml:"environment"`
	Version    string `json:"version" yaml:"version"`
}

func NewTracer(config TraceConfig) (*Tracer, error) {
	if !config.Enabled {
		return &Tracer{tracer: otel.Tracer("noop")}, nil
	}

	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stdouttrace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.Version),
			attribute.String("environment", config.Environment),
		)),
	)

	otel.SetTracerProvider(tp)

	return &Tracer{tracer: tp.Tracer(config.ServiceName)}, nil
}

func (t *Tracer) StartSpan(ctx context.Context, name string, attributes ...attribute.KeyValue) (context.Context, oteltrace.Span) {
	ctx, span := t.tracer.Start(ctx, name)
	if len(attributes) > 0 {
		span.SetAttributes(attributes...)
	}
	return ctx, span
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(*trace.TracerProvider); ok {
		return tp.Shutdown(ctx)
	}
	return nil
}

func DefaultTraceConfig() TraceConfig {
	return TraceConfig{
		Enabled:     false,
		Exporter:    "stdout",
		ServiceName: "go-spec-mock",
		Environment: "production",
		Version:     "1.0.0",
	}
}