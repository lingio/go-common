package common

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var tracer = otel.Tracer("lingio.com/go-common")

func InitMonitoring(serviceName string, cfg MonitorConfig) error {
	env := ParseEnv()

	if err := InitServiceTraceProvider(serviceName, env, cfg.TempoHost); err != nil {
		return fmt.Errorf("init trace provider: %w", err)
	}

	return nil
}

func InitServiceTraceProvider(serviceName string, env Environment, tempoHost string) error {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(tempoHost),
		otlptracehttp.WithInsecure(),
		// NOTE (Axel): retry config subject to change depending on service load
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: 10 * time.Second,
			MaxInterval:     5 * time.Minute,
			MaxElapsedTime:  30 * time.Minute, // if retry fails here, dump traces
		}),
	}

	// specify environment-dependent options
	if env == EnvDevelop {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	client, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(client),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(GetBuildCommitHash()),
			semconv.DeploymentEnvironmentKey.String(string(env)),
		)),
	)
	otel.SetTracerProvider(tp) // set as global trace provider
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return nil
}

func GetBuildCommitHash() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, kv := range info.Settings {
			if kv.Key == "vcs.revision" {
				return kv.Value
			}
		}
	}
	return "unknown revision"
}
