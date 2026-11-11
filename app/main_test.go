package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

var testConfig = config{AppEnv: "test", Message: "hello-from-test", APIKeySet: true}

// discard drops log records in tests that do not check them.
var discard = slog.New(slog.DiscardHandler)

// do sends one request through a mux and returns the recorded response.
func do(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
	return rec
}

func TestHealth(t *testing.T) {
	rec := do(t, newMux(testConfig), "/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestWorkReturnsMessage(t *testing.T) {
	rec := do(t, newMux(testConfig), "/api/work?ms=0")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body workResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Message != testConfig.Message {
		t.Errorf("message = %q, want %q", body.Message, testConfig.Message)
	}
}

// TestWorkReturnsVersion checks that /api/work names the build that
// answered, so a deploy is visible from curl.
func TestWorkReturnsVersion(t *testing.T) {
	old := version
	version = "abc1234"
	t.Cleanup(func() { version = old })

	var body workResponse
	if err := json.NewDecoder(do(t, newMux(testConfig), "/api/work?ms=0").Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Version != "abc1234" {
		t.Errorf("version = %q, want %q", body.Version, "abc1234")
	}
}

// The k6 check expects the configmap message on every response, failed ones
// included, so the error is added to the body rather than replacing it.
func TestWorkFailureKeepsMessage(t *testing.T) {
	rec := do(t, newMux(testConfig), "/api/work?ms=0&fail=1")
	var body workResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Message != testConfig.Message {
		t.Errorf("message = %q, want %q", body.Message, testConfig.Message)
	}
	if body.Status != http.StatusInternalServerError || body.Error != "upstream timeout after 0ms" {
		t.Errorf("status = %d, error = %q; want 500 and the upstream timeout", body.Status, body.Error)
	}
}

func TestWorkStatus(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   int
	}{
		{"fail=1 returns 500", "/api/work?ms=0&fail=1", http.StatusInternalServerError},
		{"fail=0 returns 200", "/api/work?ms=0&fail=0", http.StatusOK},
		{"fail clamped above 1", "/api/work?ms=0&fail=5", http.StatusInternalServerError},
		{"ms clamped below 0", "/api/work?ms=-10", http.StatusOK},
		{"ms not a number", "/api/work?ms=abc", http.StatusBadRequest},
		{"ms not an integer", "/api/work?ms=1.5", http.StatusBadRequest},
		{"fail not a number", "/api/work?ms=0&fail=abc", http.StatusBadRequest},
		{"fail NaN", "/api/work?ms=0&fail=NaN", http.StatusBadRequest},
	}
	h := newMux(testConfig)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := do(t, h, tt.target); rec.Code != tt.want {
				t.Errorf("GET %s: status = %d, want %d", tt.target, rec.Code, tt.want)
			}
		})
	}
}

func TestParseWorkParams(t *testing.T) {
	tests := []struct {
		query    string
		wantMS   int
		wantFail float64
	}{
		{"", defaultMS, defaultFail},
		{"ms=200", 200, 0},
		{"ms=5000", maxMS, 0},
		{"ms=-1", 0, 0},
		{"fail=0.25", defaultMS, 0.25},
		{"fail=-1", defaultMS, 0},
		{"fail=2", defaultMS, 1},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/work?"+tt.query, nil)
			ms, fail, err := parseWorkParams(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ms != tt.wantMS || fail != tt.wantFail {
				t.Errorf("got ms=%d fail=%v, want ms=%d fail=%v", ms, fail, tt.wantMS, tt.wantFail)
			}
		})
	}
}

func TestMetricsCountRequests(t *testing.T) {
	h, reader := meteredHandler(t)
	for range 5 {
		do(t, h, "/api/work?ms=0&fail=1")
	}
	for range 10 {
		do(t, h, "/api/work?ms=0")
	}
	do(t, h, "/foo/1")
	do(t, h, "/foo/2")

	rm := collect(t, reader)
	for _, tt := range []struct {
		route, status string
		want          int64
	}{
		{"/api/work", "500", 5},
		{"/api/work", "200", 10},
		{"other", "404", 2},
	} {
		if got := requestCount(t, rm, tt.route, tt.status); got != tt.want {
			t.Errorf("http.requests{route=%q,status=%q} = %d, want %d", tt.route, tt.status, got, tt.want)
		}
	}
	if got := latencyPoint(t, rm, "/api/work", "200").Count; got != 10 {
		t.Errorf("http.request.duration{route=/api/work,status=200} count = %d, want 10", got)
	}
	for _, dp := range requestPoints(t, rm) {
		if route, _ := dp.Attributes.Value("route"); strings.HasPrefix(route.AsString(), "/foo/") {
			t.Errorf("metrics contain a raw path as a route label: %q", route.AsString())
		}
	}
}

func TestMetricsCounterIncrements(t *testing.T) {
	h, reader := meteredHandler(t)

	do(t, h, "/health")
	if got := requestCount(t, collect(t, reader), "/health", "200"); got != 1 {
		t.Fatalf("after 1 request: http.requests = %d, want 1", got)
	}
	do(t, h, "/health")
	if got := requestCount(t, collect(t, reader), "/health", "200"); got != 2 {
		t.Fatalf("after 2 requests: http.requests = %d, want 2", got)
	}
}

func TestNoMetricsEndpoint(t *testing.T) {
	// Metrics are pushed to the Collector, so the app no longer serves them.
	if rec := do(t, newMux(testConfig), "/metrics"); rec.Code != http.StatusNotFound {
		t.Errorf("/metrics status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// meteredHandler returns a handler whose metrics are read on demand from an
// in-memory reader.
func meteredHandler(t *testing.T) (http.Handler, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })
	return newHandler(testConfig, tracenoop.NewTracerProvider(), mp, discard), reader
}

func collect(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &rm); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	return rm
}

// find returns the metric with the given instrument name.
func find(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %q not recorded", name)
	return metricdata.Metrics{}
}

func requestPoints(t *testing.T, rm metricdata.ResourceMetrics) []metricdata.DataPoint[int64] {
	t.Helper()
	sum, ok := find(t, rm, "http.requests").Data.(metricdata.Sum[int64])
	if !ok || !sum.IsMonotonic || sum.Temporality != metricdata.CumulativeTemporality {
		t.Fatalf("http.requests is not a cumulative monotonic sum")
	}
	return sum.DataPoints
}

func matches(set attribute.Set, route, status string) bool {
	r, _ := set.Value("route")
	s, _ := set.Value("status")
	m, _ := set.Value("method")
	return r.AsString() == route && s.AsString() == status && m.AsString() == http.MethodGet
}

// requestCount returns the http.requests value for one GET route and status.
func requestCount(t *testing.T, rm metricdata.ResourceMetrics, route, status string) int64 {
	t.Helper()
	for _, dp := range requestPoints(t, rm) {
		if matches(dp.Attributes, route, status) {
			return dp.Value
		}
	}
	return 0
}

// latencyPoint returns the http.request.duration point for one GET route and
// status.
func latencyPoint(t *testing.T, rm metricdata.ResourceMetrics, route, status string) metricdata.HistogramDataPoint[float64] {
	t.Helper()
	hist, ok := find(t, rm, "http.request.duration").Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("http.request.duration is not a histogram")
	}
	for _, dp := range hist.DataPoints {
		if matches(dp.Attributes, route, status) {
			return dp
		}
	}
	t.Fatalf("no http.request.duration point for route=%q status=%q", route, status)
	return metricdata.HistogramDataPoint[float64]{}
}
