package gothic

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	runtimeclient "github.com/metru/gothic/client/runtime"
)

// ProcedureFunc handles a server procedure call.
type ProcedureFunc func(ctx context.Context, body []byte) (any, error)

// Server is the gothic HTTP server. It wraps a chi router, serves the client
// runtime and generated modules, and dispatches procedure calls.
type Server struct {
	mux        *chi.Mux
	mu         sync.RWMutex
	procedures map[string]ProcedureFunc
}

// NewServer returns a Server with a chi router.
func NewServer() *Server {
	return &Server{
		mux:        chi.NewRouter(),
		procedures: map[string]ProcedureFunc{},
	}
}

// Get registers a GET route.
func (s *Server) Get(pattern string, h http.HandlerFunc) {
	s.mux.Get(pattern, h)
}

// Post registers a POST route.
func (s *Server) Post(pattern string, h http.HandlerFunc) {
	s.mux.Post(pattern, h)
}

// Handle registers a handler for a pattern.
func (s *Server) Handle(pattern string, h http.Handler) {
	s.mux.Handle(pattern, h)
}

// Use appends middleware to the router.
func (s *Server) Use(middlewares ...func(http.Handler) http.Handler) {
	s.mux.Use(middlewares...)
}

// RegisterProcedure registers a server procedure. Generated code calls this
// from a file-level registrar.
func (s *Server) RegisterProcedure(name string, fn ProcedureFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.procedures[name] = fn
}

// ListenAndServe starts the server and logs the URL it is listening on.
func (s *Server) ListenAndServe(addr string) error {
	log.Printf("gothic: listening on %s", displayURL(addr))
	return http.ListenAndServe(addr, s)
}

func displayURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/_gothic/runtime.js":
		serveJS(w, runtimeclient.RuntimeJS)
	case strings.HasPrefix(r.URL.Path, "/_gothic/modules/"):
		data, ok := lookupModule(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		serveJS(w, data)
	case strings.HasPrefix(r.URL.Path, "/_gothic/rpc/"):
		s.handleRPC(w, r)
	default:
		s.mux.ServeHTTP(w, r)
	}
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/_gothic/rpc/")
	s.mu.RLock()
	fn, ok := s.procedures[name]
	s.mu.RUnlock()
	if !ok {
		writeRPCError(w, http.StatusNotFound, "unknown procedure")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeRPCError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := fn(r.Context(), body)
	if err != nil {
		writeRPCError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"result": result})
}

func writeRPCError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": message},
	})
}

func serveJS(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(data)
}
