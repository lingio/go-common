package common

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var tracer = otel.Tracer("lingio.com/go-common")

func InitServiceTraceProvider(serviceName, tempoHost string) error {
	client, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(tempoHost),
		// NOTE (Axel): retry config subject to change depending on service load
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: 10 * time.Second,
			MaxInterval:     5 * time.Minute,
			MaxElapsedTime:  30 * time.Minute, // if retry fails here, dump traces
		}),
	)
	if err != nil {
		return err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(client),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)
	otel.SetTracerProvider(tp) // set as global trace provider
	return nil
}
