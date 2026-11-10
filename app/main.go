// Command devops-app is a small HTTP service that implements the Group B app
// contract: a health check and a simulated-work endpoint, with OpenTelemetry
// metrics, traces and logs pushed over OTLP to the Collector.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	addr = ":8080"
	// defaultAdminAddr serves the fault toggles (faults.go). It is in no
	// Service, so it is reachable only through kubectl port-forward.
	defaultAdminAddr = ":8081"

	defaultMS   = 50
	maxMS       = 2000
	defaultFail = 0.0

	// shutdownTimeout must exceed the longest /api/work delay so in-flight
	// requests finish when the pod receives SIGTERM.
	shutdownTimeout = 10 * time.Second
)

// config holds settings read from the environment. API_KEY is kept only as a
// flag so its value can never leak into logs or responses.
type config struct {
	AppEnv    string
	Message   string
	APIKeySet bool
	AdminAddr string
}

func loadConfig() config {
	return config{
		AppEnv:    getenv("APP_ENV", "dev"),
		Message:   getenv("MESSAGE", "hello from devops-app"),
		APIKeySet: os.Getenv("API_KEY") != "",
		AdminAddr: getenv("ADMIN_ADDR", defaultAdminAddr),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// newMux wires the routes with the global tracer and meter providers and the
// default logger.
func newMux(cfg config) http.Handler {
	return newHandler(cfg, otel.GetTracerProvider(), otel.GetMeterProvider(), slog.Default())
}

// newHandler wires the routes with fault toggles at their defaults.
func newHandler(cfg config, tp trace.TracerProvider, mp metric.MeterProvider, logger *slog.Logger) http.Handler {
	return newAppHandler(cfg, tp, mp, logger, newFaults())
}

// newAppHandler wires the routes and wraps them in RED metrics and a server
// span per request. f is shared with the admin listener.
func newAppHandler(cfg config, tp trace.TracerProvider, mp metric.MeterProvider, logger *slog.Logger, f *faults) http.Handler {
	m := newMetrics(mp)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readyHandler(f))
	mux.HandleFunc("GET /api/work", workHandler(cfg, logger, f))
	return otelhttp.NewHandler(m.instrument(mux), "http",
		otelhttp.WithTracerProvider(tp),
		// otelhttp would record its own request metrics without the route
		// label; instrument records the RED series instead.
		otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
		otelhttp.WithPropagators(propagator()),
		otelhttp.WithFilter(traced),
	)
}

// metrics holds the RED instruments from the app contract. The Collector's
// Prometheus exporter names them http_requests_total and
// http_request_duration_seconds.
type metrics struct {
	requests metric.Int64Counter
	duration metric.Float64Histogram
}

func newMetrics(mp metric.MeterProvider) *metrics {
	meter := mp.Meter("devops-app")
	requests, err := meter.Int64Counter("http.requests",
		metric.WithDescription("HTTP requests by method, route template, and status code."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}
	duration, err := meter.Float64Histogram("http.request.duration",
		metric.WithDescription("HTTP request latency by method, route template, and status code."),
		metric.WithUnit("s"),
		// Covers the full /api/work delay range (up to 2 s).
		metric.WithExplicitBucketBoundaries(0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 2.5),
	)
	if err != nil {
		otel.Handle(err)
	}
	return &metrics{requests: requests, duration: duration}
}

// instrument records every request handled by mux. It runs after mux has
// matched, so r.Pattern holds the route template. otelhttp reads the same
// pattern to name the span ("GET /api/work") but reads http.route only before
// the match, so it is set here. The request context carries the span, so the
// SDK keeps its trace ID as an exemplar on the latency sample.
func (m *metrics) instrument(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		mux.ServeHTTP(rec, r)

		// Readiness probes are kubelet traffic: a not-ready pod's 503s would
		// otherwise show up in the dashboard's error ratio.
		if r.Pattern == "GET /ready" {
			return
		}
		route := routeLabel(r.Pattern)
		if r.Pattern != "" {
			trace.SpanFromContext(r.Context()).SetAttributes(semconv.HTTPRoute(route))
		}

		attrs := metric.WithAttributes(
			attribute.String("method", methodLabel(r.Method)),
			attribute.String("route", route),
			attribute.String("status", strconv.Itoa(rec.status)),
		)
		ctx := r.Context()
		m.requests.Add(ctx, 1, attrs)
		m.duration.Record(ctx, time.Since(start).Seconds(), attrs)
	})
}

// routeLabel turns a matched pattern such as "GET /api/work" into
// "/api/work". Unmatched requests (404, 405) have no pattern and become
// "other", so raw paths never become label values.
func routeLabel(pattern string) string {
	if pattern == "" {
		return "other"
	}
	if _, path, ok := strings.Cut(pattern, " "); ok {
		return path
	}
	return pattern
}

// methodLabel keeps the method label bounded when clients send arbitrary
// methods.
func methodLabel(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return method
	}
	return "other"
}

// statusRecorder captures the status code written by the wrapped handler.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type workResponse struct {
	Message string `json:"message"`
	AppEnv  string `json:"app_env"`
	MS      int    `json:"ms"`
	Status  int    `json:"status"`
}

func workHandler(cfg config, logger *slog.Logger, f *faults) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ms, fail, err := parseWorkParams(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		// otelhttp records url.path but not the query, so without these the
		// span would show a 500 but not that the caller asked for it.
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(
			attribute.Int("work.ms", ms),
			attribute.Float64("work.fail_probability", fail),
		)
		fs := f.get()
		if fs.active() {
			span.SetAttributes(
				attribute.Float64("fault.error_rate", fs.ErrorRate),
				attribute.Int("fault.extra_latency_ms", fs.ExtraLatencyMS),
			)
		}

		select {
		case <-time.After(time.Duration(ms+fs.ExtraLatencyMS) * time.Millisecond):
		case <-r.Context().Done():
			return
		}

		status := http.StatusOK
		reason := ""
		switch {
		case rand.Float64() < fail:
			reason = "random draw below fail=" + strconv.FormatFloat(fail, 'g', -1, 64)
		case rand.Float64() < fs.ErrorRate:
			reason = "fault toggle error_rate=" + strconv.FormatFloat(fs.ErrorRate, 'g', -1, 64)
		}
		injected := reason != ""
		span.SetAttributes(attribute.Bool("work.failure_injected", injected))
		if injected {
			status = http.StatusInternalServerError
			// An event, not a status description: otelhttp sets the 5xx
			// status after the handler returns and would clear the description.
			span.AddEvent("injected_failure", trace.WithAttributes(
				attribute.String("reason", reason),
			))
			// The request context carries the span, so the OTel log handler
			// stamps this record with its trace and span IDs.
			logger.WarnContext(r.Context(), "injected failure",
				"route", "/api/work", "work.ms", ms, "work.fail_probability", fail, "reason", reason)
		}
		writeJSON(w, status, workResponse{
			Message: cfg.Message,
			AppEnv:  cfg.AppEnv,
			MS:      ms,
			Status:  status,
		})
	}
}

// parseWorkParams reads ms and fail, applying defaults and clamping
// out-of-range values. Malformed values are errors.
func parseWorkParams(r *http.Request) (int, float64, error) {
	q := r.URL.Query()

	ms := defaultMS
	if s := q.Get("ms"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, 0, errors.New("ms must be an integer")
		}
		ms = min(max(v, 0), maxMS)
	}

	fail := defaultFail
	if s := q.Get("fail"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsNaN(v) {
			return 0, 0, errors.New("fail must be a number")
		}
		fail = min(max(v, 0), 1)
	}

	return ms, fail, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write response", "err", err)
	}
}

func main() {
	if err := run(); err != nil {
		slog.Error("exiting", "err", err)
		os.Exit(1)
	}
}

// run serves until SIGTERM or SIGINT, then drains in-flight requests.
func run() error {
	cfg := loadConfig()

	shutdownTelemetry, err := setupTelemetry(context.Background())
	if err != nil {
		return err
	}
	// Logs still go to stderr for kubectl logs, and also to the OTel logger
	// provider (a no-op when no OTLP endpoint is set).
	slog.SetDefault(slog.New(slog.NewMultiHandler(
		slog.NewTextHandler(os.Stderr, nil),
		otelslog.NewHandler("devops-app"),
	)))
	defer func() {
		// Flush buffered spans and the last metric values after the server
		// has drained.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			slog.Error("flush telemetry", "err", err)
		}
	}()

	f := newFaults()
	srv := &http.Server{
		Addr:              addr,
		Handler:           newAppHandler(cfg, otel.GetTracerProvider(), otel.GetMeterProvider(), slog.Default(), f),
		ReadHeaderTimeout: 5 * time.Second,
	}
	admin := &http.Server{
		Addr:              cfg.AdminAddr,
		Handler:           newAdminMux(f, slog.Default()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	errCh := make(chan error, 2)
	go func() {
		slog.Info("starting", "addr", addr, "admin_addr", cfg.AdminAddr, "app_env", cfg.AppEnv, "api_key_set", cfg.APIKeySet)
		errCh <- srv.ListenAndServe()
	}()
	go func() {
		errCh <- admin.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	// The admin listener holds no in-flight work worth draining.
	_ = admin.Close()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("stopped")
	return nil
}
