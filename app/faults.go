package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"sync"
)

// faultState is the set of runtime fault toggles, held per pod in memory. The
// presenter flips them through the admin listener during the live demo, so
// the traffic stays the same while the failure changes.
type faultState struct {
	// ErrorRate is the chance (0..1) that /api/work returns 500, on top of
	// any ?fail= the caller asks for.
	ErrorRate float64 `json:"error_rate"`
	// ExtraLatencyMS (0..maxMS) is added to each /api/work delay.
	ExtraLatencyMS int `json:"extra_latency_ms"`
	// Ready is what /ready reports, and so whether the pod receives traffic.
	Ready bool `json:"ready"`
}

func defaultFaults() faultState {
	return faultState{Ready: true}
}

// active reports whether a toggle changes /api/work.
func (s faultState) active() bool {
	return s.ErrorRate > 0 || s.ExtraLatencyMS > 0
}

type faults struct {
	mu    sync.RWMutex
	state faultState
}

func newFaults() *faults {
	return &faults{state: defaultFaults()}
}

func (f *faults) get() faultState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

// faultUpdate is a PUT /faults body. Pointers tell a field that was left out
// from one set to its zero value.
type faultUpdate struct {
	ErrorRate      *float64 `json:"error_rate"`
	ExtraLatencyMS *int     `json:"extra_latency_ms"`
	Ready          *bool    `json:"ready"`
}

// apply clamps out-of-range values like parseWorkParams and returns the new
// state.
func (f *faults) apply(u faultUpdate) faultState {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u.ErrorRate != nil {
		f.state.ErrorRate = min(max(*u.ErrorRate, 0), 1)
	}
	if u.ExtraLatencyMS != nil {
		f.state.ExtraLatencyMS = min(max(*u.ExtraLatencyMS, 0), maxMS)
	}
	if u.Ready != nil {
		f.state.Ready = *u.Ready
	}
	return f.state
}

func (f *faults) reset() faultState {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = defaultFaults()
	return f.state
}

// newAdminMux serves the fault toggles. It runs on its own port, which is in
// no Service, so only kubectl port-forward reaches it; it has no auth.
func newAdminMux(f *faults, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /faults", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, f.get())
	})
	mux.HandleFunc("PUT /faults", func(w http.ResponseWriter, r *http.Request) {
		u, err := decodeFaultUpdate(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s := f.apply(u)
		logFaults(r, logger, "faults changed", s)
		writeJSON(w, http.StatusOK, s)
	})
	mux.HandleFunc("DELETE /faults", func(w http.ResponseWriter, r *http.Request) {
		s := f.reset()
		logFaults(r, logger, "faults reset", s)
		writeJSON(w, http.StatusOK, s)
	})
	return mux
}

func decodeFaultUpdate(r *http.Request) (faultUpdate, error) {
	var u faultUpdate
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&u); err != nil {
		return u, errors.New("body must be JSON with error_rate, extra_latency_ms or ready")
	}
	if u.ErrorRate != nil && math.IsNaN(*u.ErrorRate) {
		return u, errors.New("error_rate must be a number")
	}
	return u, nil
}

func logFaults(r *http.Request, logger *slog.Logger, msg string, s faultState) {
	logger.InfoContext(r.Context(), msg,
		"fault.error_rate", s.ErrorRate, "fault.extra_latency_ms", s.ExtraLatencyMS, "fault.ready", s.Ready)
}

// readyHandler backs the readinessProbe. A 503 takes the pod out of the
// Service endpoints without a restart, because the livenessProbe stays on
// /health.
func readyHandler(f *faults) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if !f.get().Ready {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
