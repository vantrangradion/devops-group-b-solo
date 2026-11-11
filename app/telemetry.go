package main

import (
	"context"
	"errors"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// setupTelemetry pushes spans, metrics and logs over OTLP/HTTP for each signal that
// has an OTLP endpoint set. The exporters, sampler, export interval and
// resource read the standard OTEL_* variables, so the Deployment, not the
// code, decides where telemetry goes and how often metrics are sent. A signal
// with no endpoint keeps the global no-op provider.
func setupTelemetry(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdowns []func(context.Context) error
	shutdown = func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdowns {
			errs = append(errs, fn(ctx))
		}
		return errors.Join(errs...)
	}
	// On a setup error, release whatever was already started.
	defer func() {
		if err != nil {
			err = errors.Join(err, shutdown(ctx))
		}
	}()

	// resource.Default reads OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES.
	// service.version is the build's commit, so traces, metrics and logs say
	// which deploy produced them; OTEL_RESOURCE_ATTRIBUTES can still override it.
	res, err := resource.Merge(resource.Default(),
		resource.NewSchemaless(semconv.ServiceVersion(version)))
	if err != nil {
		return shutdown, err
	}
	if res, err = resource.Merge(res, resource.Environment()); err != nil {
		return shutdown, err
	}

	if endpointSet("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") {
		exp, err := otlptracehttp.New(ctx)
		if err != nil {
			return shutdown, err
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, tp.Shutdown)
	}

	if endpointSet("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") {
		exp, err := otlpmetrichttp.New(ctx)
		if err != nil {
			return shutdown, err
		}
		// The periodic reader reads OTEL_METRIC_EXPORT_INTERVAL (default 60 s).
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, mp.Shutdown)
	}

	if endpointSet("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT") {
		exp, err := otlploghttp.New(ctx)
		if err != nil {
			return shutdown, err
		}
		lp := sdklog.NewLoggerProvider(
			sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
			sdklog.WithResource(res),
		)
		otel.SetLoggerProvider(lp)
		shutdowns = append(shutdowns, lp.Shutdown)
	}

	return shutdown, nil
}

// endpointSet reports whether a signal has an OTLP endpoint, either its own or
// the shared one.
func endpointSet(signalEnv string) bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv(signalEnv) != ""
}

// propagator reads and writes the W3C traceparent and baggage headers, so a
// trace started upstream (k6, a browser) continues in this app.
func propagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	)
}

// traced reports whether a request gets a span. Probes hit /health and /ready
// every few seconds and would bury the /api/work traces.
func traced(r *http.Request) bool {
	return r.URL.Path != "/health" && r.URL.Path != "/ready"
}
