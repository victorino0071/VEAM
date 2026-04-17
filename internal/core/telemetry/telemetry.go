package telemetry

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	once sync.Once
)

// InitTelemetry inicializa o SDK do OpenTelemetry e o Logger Estruturado.
func InitTelemetry(serviceName string) (func(context.Context) error, error) {
	var shutdown func(context.Context) error
	var err error

	once.Do(func() {
		// 1. Configura o JSON Logger (slog)
		// Em produção, isso seria filtrado por Level (Info/Error)
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		slog.SetDefault(slog.New(handler))

		// 2. Configura o OpenTelemetry (Exportador p/ Console p/ demonstração)
		exporter, expErr := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if expErr != nil {
			err = expErr
			return
		}

		res, resErr := resource.Merge(
			resource.Default(),
			resource.NewWithAttributes(
				"", // Deixa em branco para herdar/evitar conflito de SchemaURL
				semconv.ServiceName(serviceName),
			),
		)
		if resErr != nil {
			err = resErr
			return
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)

		shutdown = tp.Shutdown
	})

	return shutdown, err
}
