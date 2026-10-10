package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testConfig = config{AppEnv: "test", Message: "hello-from-test", APIKeySet: true}

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
	h := newMux(testConfig)
	for range 5 {
		do(t, h, "/api/work?ms=0&fail=1")
	}
	for range 10 {
		do(t, h, "/api/work?ms=0")
	}
	do(t, h, "/foo/1")
	do(t, h, "/foo/2")

	body := scrape(t, h)
	for _, want := range []string{
		`http_requests_total{method="GET",route="/api/work",status="500"} 5`,
		`http_requests_total{method="GET",route="/api/work",status="200"} 10`,
		`http_requests_total{method="GET",route="other",status="404"} 2`,
		`http_request_duration_seconds_count{method="GET",route="/api/work",status="200"} 10`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics missing %q", want)
		}
	}
	if strings.Contains(body, `route="/foo/`) {
		t.Error("metrics contain a raw path as a route label")
	}
}

func TestMetricsCounterIncrements(t *testing.T) {
	h := newMux(testConfig)
	const series = `http_requests_total{method="GET",route="/health",status="200"}`

	do(t, h, "/health")
	if body := scrape(t, h); !strings.Contains(body, series+" 1") {
		t.Fatalf("after 1 request: metrics missing %q", series+" 1")
	}
	do(t, h, "/health")
	if body := scrape(t, h); !strings.Contains(body, series+" 2") {
		t.Fatalf("after 2 requests: metrics missing %q", series+" 2")
	}
}

// scrape returns the /metrics body.
func scrape(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := do(t, h, "/metrics")
	if rec.Code != http.StatusOK {
		t.Fatalf("/metrics status = %d, want %d", rec.Code, http.StatusOK)
	}
	b, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read /metrics body: %v", err)
	}
	return string(b)
}
