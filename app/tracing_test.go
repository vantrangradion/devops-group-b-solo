package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// tracedHandler returns a handler whose spans land in an in-memory recorder.
func tracedHandler(t *testing.T) (http.Handler, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
	return newHandler(testConfig, tp), sr
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

func TestTraceSkipsProbesAndScrapes(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/health")
	do(t, h, "/metrics")

	if n := len(sr.Ended()); n != 0 {
		t.Errorf("spans = %d, want 0 for /health and /metrics", n)
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

func TestMetricsExemplar(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/api/work?ms=0")
	traceID := sr.Ended()[0].SpanContext().TraceID().String()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	req.Header.Set("Accept", "application/openmetrics-text; version=1.0.0")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body, _ := io.ReadAll(rec.Body)

	if want := `trace_id="` + traceID + `"`; !strings.Contains(string(body), want) {
		t.Errorf("OpenMetrics output missing exemplar %s", want)
	}
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
	if v, ok := attr(s, "work.failure_injected"); !ok || v.AsBool() {
		t.Errorf("work.failure_injected = %v, want false", v.AsBool())
	}
	if n := len(s.Events()); n != 0 {
		t.Errorf("events = %d, want 0 for a request that did not fail", n)
	}
}

func TestTraceInjectedFailure(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/api/work?ms=0&fail=1")

	s := sr.Ended()[0]
	if v, ok := attr(s, "work.fail_probability"); !ok || v.AsFloat64() != 1 {
		t.Errorf("work.fail_probability = %v, want 1", v.AsFloat64())
	}
	if v, ok := attr(s, "work.failure_injected"); !ok || !v.AsBool() {
		t.Errorf("work.failure_injected = %v, want true", v.AsBool())
	}
	events := s.Events()
	if len(events) != 1 || events[0].Name != "injected_failure" {
		t.Fatalf("events = %v, want one injected_failure", events)
	}
	want := "random draw below fail=1"
	for _, kv := range events[0].Attributes {
		if kv.Key == "reason" && kv.Value.AsString() == want {
			return
		}
	}
	t.Errorf("injected_failure event has no reason %q", want)
}
