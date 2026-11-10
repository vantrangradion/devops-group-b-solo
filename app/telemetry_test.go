package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log/logtest"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// tracedHandler returns a handler whose spans land in an in-memory recorder.
func tracedHandler(t *testing.T) (http.Handler, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
	return newHandler(testConfig, tp, metricnoop.NewMeterProvider(), discard), sr
}

func attr(span sdktrace.ReadOnlySpan, key attribute.Key) (attribute.Value, bool) {
	for _, kv := range span.Attributes() {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestTraceServerSpan(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/api/work?ms=0")

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	s := spans[0]
	if s.Name() != "GET /api/work" {
		t.Errorf("name = %q, want %q", s.Name(), "GET /api/work")
	}
	if s.SpanKind() != trace.SpanKindServer {
		t.Errorf("kind = %v, want server", s.SpanKind())
	}
	if v, ok := attr(s, "http.route"); !ok || v.AsString() != "/api/work" {
		t.Errorf("http.route = %q, want /api/work", v.AsString())
	}
	if v, ok := attr(s, "http.response.status_code"); !ok || v.AsInt64() != http.StatusOK {
		t.Errorf("http.response.status_code = %d, want 200", v.AsInt64())
	}
	if s.Status().Code == codes.Error {
		t.Error("status = Error for a 200 response")
	}
}

func TestTraceErrorStatus(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/api/work?ms=0&fail=1")

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	if got := spans[0].Status().Code; got != codes.Error {
		t.Errorf("status = %v, want Error for a 500 response", got)
	}
}

func TestTraceSkipsProbes(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/health")

	if n := len(sr.Ended()); n != 0 {
		t.Errorf("spans = %d, want 0 for /health", n)
	}
}

func TestTraceContinuesIncomingTrace(t *testing.T) {
	h, sr := tracedHandler(t)

	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/work?ms=0", nil)
	req.Header.Set("traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
	h.ServeHTTP(httptest.NewRecorder(), req)

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}
	if got := spans[0].SpanContext().TraceID().String(); got != traceID {
		t.Errorf("trace id = %s, want %s from traceparent", got, traceID)
	}
}

// TestMetricsExemplar checks that a latency sample keeps the trace ID of the
// request's span, which the Collector exports as a trace_id exemplar.
func TestMetricsExemplar(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	h := newHandler(testConfig, tp, mp, discard)

	do(t, h, "/api/work?ms=0")
	traceID := sr.Ended()[0].SpanContext().TraceID()

	for _, ex := range latencyPoint(t, collect(t, reader), "/api/work", "200").Exemplars {
		if trace.TraceID(ex.TraceID) == traceID {
			return
		}
	}
	t.Errorf("latency point has no exemplar for trace %s", traceID)
}

func TestTraceWorkParams(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/api/work?ms=0")

	s := sr.Ended()[0]
	if v, ok := attr(s, "work.ms"); !ok || v.AsInt64() != 0 {
		t.Errorf("work.ms = %d, want 0", v.AsInt64())
	}
	if v, ok := attr(s, "work.fail_probability"); !ok || v.AsFloat64() != 0 {
		t.Errorf("work.fail_probability = %v, want 0", v.AsFloat64())
	}
	if n := len(s.Events()); n != 0 {
		t.Errorf("events = %d, want 0 for a request that did not fail", n)
	}
}

func TestTraceRecordsError(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/api/work?ms=0&fail=1")

	s := sr.Ended()[0]
	if s.Status().Code != codes.Error {
		t.Errorf("status = %v, want Error", s.Status().Code)
	}
	if _, ok := attr(s, "work.failure_injected"); ok {
		t.Error("work.failure_injected is set; the error path must not know why it failed")
	}
	events := s.Events()
	if len(events) != 1 || events[0].Name != "exception" {
		t.Fatalf("events = %v, want one exception", events)
	}
	got := map[string]string{}
	for _, kv := range events[0].Attributes {
		got[string(kv.Key)] = kv.Value.AsString()
	}
	if want := "upstream timeout after 0ms"; got["exception.message"] != want {
		t.Errorf("exception.message = %q, want %q", got["exception.message"], want)
	}
	if got["exception.type"] == "" {
		t.Error("exception.type is empty")
	}
}

// TestLogCorrelatedToTrace checks that a failed request writes a log
// record that carries the request's trace and span IDs.
func TestLogCorrelatedToTrace(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	rec := logtest.NewRecorder()
	logger := slog.New(otelslog.NewHandler("devops-app", otelslog.WithLoggerProvider(rec)))
	h := newHandler(testConfig, tp, metricnoop.NewMeterProvider(), logger)

	do(t, h, "/api/work?ms=0&fail=1")
	span := sr.Ended()[0].SpanContext()

	var found bool
	for _, records := range rec.Result() {
		for _, r := range records {
			if r.Body.AsString() != "work failed" {
				continue
			}
			found = true
			sc := trace.SpanContextFromContext(r.Context)
			if sc.TraceID() != span.TraceID() || sc.SpanID() != span.SpanID() {
				t.Errorf("log trace/span = %s/%s, want %s/%s", sc.TraceID(), sc.SpanID(), span.TraceID(), span.SpanID())
			}
		}
	}
	if !found {
		t.Fatal("no work failed log record")
	}
}

func TestNoLogWithoutFailure(t *testing.T) {
	rec := logtest.NewRecorder()
	logger := slog.New(otelslog.NewHandler("devops-app", otelslog.WithLoggerProvider(rec)))
	h := newHandler(testConfig, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider(), logger)

	do(t, h, "/api/work?ms=0")
	for _, records := range rec.Result() {
		if len(records) != 0 {
			t.Errorf("log records = %d, want 0 for a request that did not fail", len(records))
		}
	}
}
