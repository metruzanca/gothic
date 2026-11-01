package gothic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Recommended health endpoint paths. Health checks are opt-in; pass these to
// WithHealth and WithReadiness to enable them.
const (
	HealthPath = "/healthz"
	ReadyPath  = "/readyz"
)

const readinessTimeout = 5 * time.Second

// WithHealth sets the liveness endpoint path. An empty path disables it.
func WithHealth(path string) Option {
	return func(s *Server) { s.healthPath = path }
}

// WithReadiness sets the readiness endpoint path and an optional check. An
// empty path disables it. The check receives a request-scoped context with a
// five second deadline.
func WithReadiness(path string, check func(context.Context) error) Option {
	return func(s *Server) {
		s.readyPath = path
		s.readyCheck = check
	}
}

// serveHealth handles the health endpoints before the middleware chain, so
// auth, CSRF, and access logging do not apply. It reports whether it handled
// the request.
func (s *Server) serveHealth(w http.ResponseWriter, r *http.Request) bool {
	ready := false
	switch {
	case s.healthPath != "" && r.URL.Path == s.healthPath:
	case s.readyPath != "" && r.URL.Path == s.readyPath:
		ready = true
	default:
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeSpecialJSON(w, r, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return true
	}
	if ready && s.readyCheck != nil {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()
		if err := s.runReadyCheck(ctx); err != nil {
			s.loggerOrDefault().Warn("readiness check failed", "error", err)
			writeSpecialJSON(w, r, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return true
		}
	}
	writeSpecialJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
	return true
}

// runReadyCheck converts a panic in a readiness check into a failed check.
func (s *Server) runReadyCheck(ctx context.Context) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("readiness check panic: %v", recovered)
		}
	}()
	return s.readyCheck(ctx)
}

// writeSpecialJSON writes a small, uncached JSON response. HEAD requests get
// the headers without a body.
func writeSpecialJSON(w http.ResponseWriter, r *http.Request, status int, value any) {
	header := w.Header()
	header.Set("Content-Type", "application/json; charset=utf-8")
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}
