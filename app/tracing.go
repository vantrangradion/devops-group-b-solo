package main

import (
	"context"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// setupTracing exports spans over OTLP/HTTP when an OTLP endpoint is set. The
// exporter, sampler and resource read the standard OTEL_* variables, so the
// Deployment, not the code, decides where spans go. With no endpoint the
// global provider stays the no-op one and the app behaves as before.
func setupTracing(ctx context.Context) (shutdown func(context.Context) error, err error) {
	noop := func(context.Context) error { return nil }
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" &&
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		return noop, nil
	}

	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return noop, err
	}
	// resource.Default reads OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES.
	res, err := resource.Merge(resource.Default(), resource.Environment())
	if err != nil {
		return noop, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// propagator reads and writes the W3C traceparent and baggage headers, so a
// trace started upstream (k6, a browser) continues in this app.
func propagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	)
}

// traced reports whether a request gets a span. Probes and scrapes hit
// /health and /metrics every few seconds and would bury the /api/work traces.
func traced(r *http.Request) bool {
	return r.URL.Path != "/health" && r.URL.Path != "/metrics"
}
