package common

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	gcptraceexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
)

var tracer = otel.Tracer("lingio.com/go-common")

// traceconfig contains both tracesdk and otel options.
// There is currently no nice way of deserializing json into otel/tracesdk data
// structures, otherwise we could have specified this in service config instead.
type traceconfig struct {
	Retry   otlptracehttp.RetryConfig
	Sampler sdktrace.Sampler
}

var traceConfigDevelop = traceconfig{
	Sampler: sdktrace.AlwaysSample(),
	Retry: otlptracehttp.RetryConfig{
		Enabled:         true,
		InitialInterval: 2 * time.Second,
		MaxInterval:     5 * time.Second,
		MaxElapsedTime:  10 * time.Second,
	},
}

var traceConfigStaging = traceconfig{
	Sampler: sdktrace.AlwaysSample(),
	Retry: otlptracehttp.RetryConfig{
		Enabled:         true,
		InitialInterval: 2 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  10 * time.Minute,
	},
}

var traceConfigProduction = traceconfig{
	// sample ~10% of traces, and try to keep distributed trace chains sampled
	Sampler: sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1)),
	Retry:   traceConfigStaging.Retry,
}

func InitMonitoring(serviceName string, cfg MonitorConfig) error {
	env := ParseEnv()

	if cfg.TempoHost != "" {
		if err := InitHttpTrace(serviceName, env, cfg.TempoHost); err != nil {
			return fmt.Errorf("init trace provider: %w", err)
		}
		return nil
	}

	if cfg.CloudTrace.Enabled {
		return InitGoogleCloudTrace(context.TODO(), serviceName, env)
	}

	return nil
}

func InitGoogleCloudTrace(ctx context.Context, serviceName string, env Environment) error {
	exporter, err := gcptraceexporter.New() // autodetect projectID from default credentials
	if err != nil {
		return Errorf(err, "gcp trace exporter")
	}

	res, err := resource.New(ctx,
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(GetBuildCommitHash()),
		),
	)
	if err != nil {
		return Errorf(err, "resource setup")
	}

	return setupTraceProvider(env, exporter, res)
}

func setupTraceProvider(env Environment, spanExporter sdktrace.SpanExporter, res *resource.Resource) error {
	// specify environment-dependent options
	var cfg traceconfig
	switch env {
	case EnvDevelop:
		cfg = traceConfigDevelop
	case EnvStaging:
		cfg = traceConfigStaging
	case EnvProduction:
		cfg = traceConfigProduction
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(cfg.Sampler),
		sdktrace.WithBatcher(spanExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp) // set as global trace provider
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return nil
}

func InitHttpTrace(serviceName string, env Environment, tempoHost string) error {
	var cfg traceconfig

	// specify environment-dependent options
	switch env {
	case EnvDevelop:
		cfg = traceConfigDevelop
	case EnvStaging:
		cfg = traceConfigStaging
	case EnvProduction:
		cfg = traceConfigProduction
	}

	client, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(tempoHost),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithRetry(cfg.Retry),
	)
	if err != nil {
		return err
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(GetBuildCommitHash()),
	)
	return setupTraceProvider(env, client, res)
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
