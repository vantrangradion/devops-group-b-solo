// Command devops-app is a small HTTP service that implements the Group B app
// contract: a health check, a simulated-work endpoint, Prometheus metrics, and
// OpenTelemetry traces.
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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	addr = ":8080"

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
}

func loadConfig() config {
	return config{
		AppEnv:    getenv("APP_ENV", "dev"),
		Message:   getenv("MESSAGE", "hello from devops-app"),
		APIKeySet: os.Getenv("API_KEY") != "",
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// newMux wires the routes with the global tracer provider.
func newMux(cfg config) http.Handler {
	return newHandler(cfg, otel.GetTracerProvider())
}

// newHandler wires the routes and wraps them in RED metrics and a server span
// per request. Each call uses its own registry, so every instance starts with
// zeroed metrics.
func newHandler(cfg config, tp trace.TracerProvider) http.Handler {
	m := newMetrics()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/work", workHandler(cfg))
	// OpenMetrics is the only format that carries exemplars. Prometheus asks
	// for it; a plain curl still gets the text format.
	mux.Handle("GET /metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	return otelhttp.NewHandler(m.instrument(mux), "http",
		otelhttp.WithTracerProvider(tp),
		otelhttp.WithPropagators(propagator()),
		otelhttp.WithFilter(traced),
	)
}

// metrics holds the RED series from the app contract.
type metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func newMetrics() *metrics {
	labels := []string{"method", "route", "status"}
	m := &metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template, and status code.",
		}, labels),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request latency by method, route template, and status code.",
			// Covers the full /api/work delay range (up to 2 s).
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 2.5},
		}, labels),
	}
	m.registry.MustRegister(
		m.requests,
		m.duration,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}

// instrument records every request handled by mux. It runs after mux has
// matched, so r.Pattern holds the route template. otelhttp reads the same
// pattern to name the span ("GET /api/work") but reads http.route only before
// the match, so it is set here, and the latency sample is linked to the
// span's trace as an exemplar.
func (m *metrics) instrument(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		mux.ServeHTTP(rec, r)

		route := routeLabel(r.Pattern)
		labels := prometheus.Labels{
			"method": methodLabel(r.Method),
			"route":  route,
			"status": strconv.Itoa(rec.status),
		}
		m.requests.With(labels).Inc()

		span := trace.SpanFromContext(r.Context())
		if r.Pattern != "" {
			span.SetAttributes(semconv.HTTPRoute(route))
		}

		elapsed := time.Since(start).Seconds()
		obs := m.duration.With(labels)
		sc := span.SpanContext()
		if eo, ok := obs.(prometheus.ExemplarObserver); ok && sc.IsSampled() {
			eo.ObserveWithExemplar(elapsed, prometheus.Labels{"trace_id": sc.TraceID().String()})
			return
		}
		obs.Observe(elapsed)
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

func workHandler(cfg config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ms, fail, err := parseWorkParams(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
		case <-r.Context().Done():
			return
		}

		status := http.StatusOK
		if rand.Float64() < fail {
			status = http.StatusInternalServerError
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

	shutdownTracing, err := setupTracing(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		// Flush buffered spans after the server has drained.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(ctx); err != nil {
			slog.Error("flush traces", "err", err)
		}
	}()

	srv := &http.Server{
		Addr:              addr,
		Handler:           newMux(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting", "addr", addr, "app_env", cfg.AppEnv, "api_key_set", cfg.APIKeySet)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("stopped")
	return nil
}
