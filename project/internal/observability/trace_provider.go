package observability

import (
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"os"
)

func ConfigureTraceProvider() *tracesdk.TracerProvider {
	jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT")
	if jaegerEndpoint == "" {
		jaegerEndpoint = fmt.Sprintf("%s/jaeger-api/api/traces", os.Getenv("GATEWAY_ADDR"))
	}

	exp, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(jaegerEndpoint),
		),
	)
	if err != nil {
		panic(err)
	}

	tp := tracesdk.NewTracerProvider(
		// WARNING: `tracesdk.WithSyncer` should be not used in production.
		// For production, you should use `tracesdk.WithBatcher`.
		tracesdk.WithSyncer(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("tickets"),
		)),
	)

	otel.SetTracerProvider(tp)

	// Don't forget this line! Omitting it will cause the trace to not be propagated via messages.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp
}
