package gothic

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

type requestIDKey struct{}

type trustedProxyKey struct{}

// trustedProxy reports whether a trusted proxy forwarded the request.
func trustedProxy(r *http.Request) bool {
	trusted, _ := r.Context().Value(trustedProxyKey{}).(bool)
	return trusted
}

// RequestID returns the request identifier set by the default middleware. It
// is empty when the middleware is disabled.
func RequestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey{}).(string)
	return id
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 128 {
			id = newRequestID()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(b[:])
}

// recoverer turns a panic into a 500 response and logs the stack.
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				err := Internal("internal server error", panicError{recovered})
				s.logError(r, err)
				s.logger.Error("panic",
					"method", r.Method,
					"path", r.URL.Path,
					"request_id", RequestID(r),
					"panic", recovered,
					"stack", string(debug.Stack()),
				)
				WriteError(w, r, err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type panicError struct{ value any }

func (p panicError) Error() string {
	if err, ok := p.value.(error); ok {
		return err.Error()
	}
	if message, ok := p.value.(string); ok {
		return message
	}
	return "panic"
}

// accessLog logs one line per request with the status, size, and duration.
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", rec.bytes,
			"duration", time.Since(start).Round(time.Microsecond).String(),
			"request_id", RequestID(r),
		)
	})
}

// realIP rewrites r.RemoteAddr from forwarding headers when the peer is a
// trusted proxy.
func (s *Server) realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.isTrustedProxy(r.RemoteAddr) {
			if ip := forwardedIP(r); ip != "" {
				r.RemoteAddr = ip
			}
			r = r.WithContext(context.WithValue(r.Context(), trustedProxyKey{}, true))
		}
		next.ServeHTTP(w, r)
	})
}

// timeoutMiddleware bounds the request context. Handlers must honor the
// context to observe the deadline.
func timeoutMiddleware(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func forwardedIP(r *http.Request) string {
	if value := headerValue(r, "X-Forwarded-For"); value != "" {
		return value
	}
	return headerValue(r, "X-Real-Ip")
}

func (s *Server) isTrustedProxy(remoteAddr string) bool {
	if len(s.trustedProxies) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, network := range s.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// responseRecorder captures the status and byte count for the access log while
// preserving the optional interfaces net/http handlers rely on.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += int64(n)
	return n, err
}

func (w *responseRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (w *responseRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// logError logs 5xx errors with the request context.
func (s *Server) logError(r *http.Request, err error) {
	if err == nil {
		return
	}
	e := asError(err)
	if e.status() < http.StatusInternalServerError {
		return
	}
	cause := e.Err
	if cause == nil {
		cause = err
	}
	s.logger.Error("request error",
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", RequestID(r),
		"error", cause,
	)
}

func (s *Server) loggerOrDefault() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.Default()
}
