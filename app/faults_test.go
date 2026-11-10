package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// faultedHandler returns the app handler and its admin handler sharing one
// fault state, with spans in an in-memory recorder.
func faultedHandler(t *testing.T) (app, admin http.Handler, sr *tracetest.SpanRecorder) {
	t.Helper()
	sr = tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
	f := newFaults()
	return newAppHandler(testConfig, tp, metricnoop.NewMeterProvider(), discard, f), newAdminMux(f, discard), sr
}

// adminDo sends one request to the admin handler and decodes the state it
// returns.
func adminDo(t *testing.T, h http.Handler, method, body string) (int, faultState) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/faults", strings.NewReader(body)))
	var s faultState
	if rec.Code == http.StatusOK {
		if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
			t.Fatalf("decode state: %v", err)
		}
	}
	return rec.Code, s
}

func TestFaultDefaults(t *testing.T) {
	_, admin, _ := faultedHandler(t)
	code, s := adminDo(t, admin, http.MethodGet, "")
	if code != http.StatusOK || s != defaultFaults() {
		t.Fatalf("GET /faults = %d %+v, want 200 %+v", code, s, defaultFaults())
	}
}

func TestFaultErrorRate(t *testing.T) {
	app, admin, sr := faultedHandler(t)
	adminDo(t, admin, http.MethodPut, `{"error_rate": 1}`)

	if rec := do(t, app, "/api/work?ms=0"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 with error_rate=1", rec.Code)
	}
	s := sr.Ended()[0]
	if v, ok := attr(s, "fault.error_rate"); !ok || v.AsFloat64() != 1 {
		t.Errorf("fault.error_rate = %v, want 1", v.AsFloat64())
	}
	events := s.Events()
	if len(events) != 1 || events[0].Name != "injected_failure" {
		t.Fatalf("events = %v, want one injected_failure", events)
	}
	for _, kv := range events[0].Attributes {
		if kv.Key == "reason" && kv.Value.AsString() == "fault toggle error_rate=1" {
			return
		}
	}
	t.Errorf("injected_failure event has no toggle reason: %v", events[0].Attributes)
}

func TestFaultExtraLatency(t *testing.T) {
	app, admin, sr := faultedHandler(t)
	adminDo(t, admin, http.MethodPut, `{"extra_latency_ms": 60}`)

	start := time.Now()
	if rec := do(t, app, "/api/work?ms=0"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if d := time.Since(start); d < 60*time.Millisecond {
		t.Errorf("duration = %v, want at least 60ms", d)
	}
	if v, ok := attr(sr.Ended()[0], "fault.extra_latency_ms"); !ok || v.AsInt64() != 60 {
		t.Errorf("fault.extra_latency_ms = %v, want 60", v.AsInt64())
	}
}

func TestFaultReady(t *testing.T) {
	app, admin, _ := faultedHandler(t)
	if rec := do(t, app, "/ready"); rec.Code != http.StatusOK {
		t.Fatalf("/ready = %d, want 200 by default", rec.Code)
	}

	adminDo(t, admin, http.MethodPut, `{"ready": false}`)
	if rec := do(t, app, "/ready"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/ready = %d, want 503 with ready=false", rec.Code)
	}
	if rec := do(t, app, "/health"); rec.Code != http.StatusOK {
		t.Errorf("/health = %d, want 200 with ready=false, so liveness never restarts the pod", rec.Code)
	}
}

func TestFaultPutClamps(t *testing.T) {
	tests := []struct {
		body string
		want faultState
	}{
		{`{"error_rate": 5}`, faultState{ErrorRate: 1, Ready: true}},
		{`{"error_rate": -1}`, faultState{Ready: true}},
		{`{"extra_latency_ms": 9999}`, faultState{ExtraLatencyMS: maxMS, Ready: true}},
		{`{"extra_latency_ms": -5}`, faultState{Ready: true}},
		{`{"error_rate": 0.3, "extra_latency_ms": 500}`, faultState{ErrorRate: 0.3, ExtraLatencyMS: 500, Ready: true}},
	}
	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			_, admin, _ := faultedHandler(t)
			code, s := adminDo(t, admin, http.MethodPut, tt.body)
			if code != http.StatusOK || s != tt.want {
				t.Errorf("PUT %s = %d %+v, want 200 %+v", tt.body, code, s, tt.want)
			}
		})
	}
}

func TestFaultPutKeepsOtherFields(t *testing.T) {
	_, admin, _ := faultedHandler(t)
	adminDo(t, admin, http.MethodPut, `{"error_rate": 0.5}`)
	_, s := adminDo(t, admin, http.MethodPut, `{"extra_latency_ms": 100}`)
	if s.ErrorRate != 0.5 || s.ExtraLatencyMS != 100 {
		t.Errorf("state = %+v, want error_rate 0.5 kept next to extra_latency_ms 100", s)
	}
}

func TestFaultPutRejectsBadBody(t *testing.T) {
	for _, body := range []string{``, `not json`, `{"error_rate": "high"}`, `{"cpu": 1}`} {
		_, admin, _ := faultedHandler(t)
		if code, _ := adminDo(t, admin, http.MethodPut, body); code != http.StatusBadRequest {
			t.Errorf("PUT %q = %d, want 400", body, code)
		}
	}
}

func TestFaultDeleteResets(t *testing.T) {
	_, admin, _ := faultedHandler(t)
	adminDo(t, admin, http.MethodPut, `{"error_rate": 1, "extra_latency_ms": 100, "ready": false}`)
	code, s := adminDo(t, admin, http.MethodDelete, "")
	if code != http.StatusOK || s != defaultFaults() {
		t.Errorf("DELETE /faults = %d %+v, want 200 %+v", code, s, defaultFaults())
	}
}

// TestNoFaultAttributesByDefault keeps spans unchanged while no toggle is on.
func TestNoFaultAttributesByDefault(t *testing.T) {
	app, _, sr := faultedHandler(t)
	do(t, app, "/api/work?ms=0")
	if _, ok := attr(sr.Ended()[0], "fault.error_rate"); ok {
		t.Error("fault.error_rate set with no toggle active")
	}
}

// TestFaultsNotOnAppPort keeps the toggles off the port the Services expose.
func TestFaultsNotOnAppPort(t *testing.T) {
	app := newHandler(testConfig, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider(), discard)
	if rec := do(t, app, "/faults"); rec.Code != http.StatusNotFound {
		t.Errorf("/faults on the app port = %d, want 404", rec.Code)
	}
}

func TestReadyNotTraced(t *testing.T) {
	h, sr := tracedHandler(t)
	do(t, h, "/ready")
	if n := len(sr.Ended()); n != 0 {
		t.Errorf("spans = %d, want 0 for /ready", n)
	}
}

// TestReadyNotCounted keeps kubelet readiness probes, and a not-ready pod's
// 503s, out of the dashboard's request rate and error ratio.
func TestReadyNotCounted(t *testing.T) {
	h, reader := meteredHandler(t)
	do(t, h, "/ready")
	do(t, h, "/health")
	rm := collect(t, reader)
	if n := requestCount(t, rm, "/ready", "200"); n != 0 {
		t.Errorf("http.requests{route=/ready} = %d, want 0", n)
	}
	if n := requestCount(t, rm, "/health", "200"); n != 1 {
		t.Errorf("http.requests{route=/health} = %d, want 1", n)
	}
}
