package otel

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config controls OTel initialization.
type Config struct {
	ServiceName    string
	ServiceVersion string
	// UseStdout enables stdout trace exporter (suitable for local dev/tests).
	UseStdout bool
}

// Init configures a global tracer provider and returns a shutdown func.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "orch"
	}
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = os.Getenv("ORCH_VERSION")
	}

	res, err := sdkresource.New(ctx,
		sdkresource.WithFromEnv(),
		sdkresource.WithProcess(),
		sdkresource.WithOS(),
		sdkresource.WithHost(),
		sdkresource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		return nil, err
	}

	var tp *sdktrace.TracerProvider
	if cfg.UseStdout {
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp,
				sdktrace.WithMaxExportBatchSize(512),
				sdktrace.WithBatchTimeout(200*time.Millisecond),
			),
			sdktrace.WithResource(res),
		)
	} else {
		// No-op exporter for now; can be extended to OTLP.
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	}

	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
