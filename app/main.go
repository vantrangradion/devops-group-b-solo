// Command devops-app is a small HTTP service that implements the Group B app
// contract: a health check, a simulated-work endpoint, and Prometheus metrics.
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
	"syscall"
	"time"
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

func newMux(cfg config) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/work", workHandler(cfg))
	return mux
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
