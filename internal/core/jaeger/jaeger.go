package jaeger

import (
	"context"

	"github.com/alexbro4u/gotemplate/internal/config"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

type EchoServer interface {
	Echo() *echo.Echo
}

type TracerProvider struct {
	tp *trace.TracerProvider
}

func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.tp != nil {
		return tp.tp.Shutdown(ctx)
	}
	return nil
}

func New(cfg config.Jaeger, httpServer EchoServer) (*TracerProvider, error) {
	if cfg.URL == "" {
		return nil, nil //nolint:nilnil // nil TracerProvider means tracing is disabled
	}

	var options []otlptracehttp.Option
	options = append(options, otlptracehttp.WithEndpointURL(cfg.URL))

	if cfg.Insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), options...)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.AppName),
		)),
	)

	otel.SetTracerProvider(tp)

	httpServer.Echo().Use(otelecho.Middleware(cfg.AppName))

	return &TracerProvider{tp: tp}, nil
}
