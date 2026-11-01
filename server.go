package gothic

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	runtimeclient "github.com/metruzanca/gothic/client/runtime"
)

// Framework paths. Generated code and the client runtime share these.
const (
	frameworkPrefix = "/_gothic"
	RuntimePath     = frameworkPrefix + "/runtime.js"
	ModulesPrefix   = frameworkPrefix + "/modules/"
	RPCPrefix       = frameworkPrefix + "/rpc/"
)

// ProcedureFunc handles a server procedure call.
type ProcedureFunc func(ctx context.Context, body []byte) (any, error)

// Server is the gothic HTTP server. It wraps a chi router, serves the client
// runtime and generated modules, dispatches procedure calls, and runs the
// global middleware chain.
type Server struct {
	mux        *chi.Mux
	mu         sync.RWMutex
	procedures map[string]ProcedureFunc

	docMu sync.RWMutex
	doc   Document

	// Runtime configuration, set by Option values in NewServer.
	logger            *slog.Logger
	addr              string
	maxBodyBytes      int64
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	shutdownTimeout   time.Duration
	requestTimeout    time.Duration
	trustedProxies    []*net.IPNet

	useDefaultMiddleware bool
	extraMiddleware      []func(http.Handler) http.Handler
	csrf                 *CSRF

	healthPath string
	readyPath  string
	readyCheck func(context.Context) error

	chainMu    sync.Mutex
	middleware []func(http.Handler) http.Handler
	chain      http.Handler

	srvMu sync.Mutex
	srv   *http.Server

	// Scope state. A child shares the root mux and module registry and owns
	// its procedures, document defaults, and middleware.
	root     *Server
	basePath string
	scopesMu sync.RWMutex
	scopes   []*Server
}

// NewServer returns a Server with a chi router and the default middleware
// stack. Options customize the server before any route is registered.
func NewServer(opts ...Option) *Server {
	s := &Server{
		mux:                  chi.NewRouter(),
		procedures:           map[string]ProcedureFunc{},
		doc:                  DefaultDocument(),
		logger:               slog.Default(),
		addr:                 ":8080",
		maxBodyBytes:         DefaultMaxBodyBytes,
		readHeaderTimeout:    5 * time.Second,
		readTimeout:          30 * time.Second,
		writeTimeout:         30 * time.Second,
		idleTimeout:          120 * time.Second,
		shutdownTimeout:      15 * time.Second,
		useDefaultMiddleware: true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	s.middleware = s.buildMiddleware()
	return s
}

func (s *Server) buildMiddleware() []func(http.Handler) http.Handler {
	var chain []func(http.Handler) http.Handler
	if s.useDefaultMiddleware {
		chain = append(chain, withRequestID)
		if len(s.trustedProxies) > 0 {
			chain = append(chain, s.realIP)
		}
		chain = append(chain, s.accessLog, s.recoverer)
	}
	if s.csrf != nil {
		chain = append(chain, s.csrf.Middleware)
	}
	if s.requestTimeout > 0 {
		chain = append(chain, timeoutMiddleware(s.requestTimeout))
	}
	chain = append(chain, s.extraMiddleware...)
	return chain
}

// SetDocument replaces the server-wide document defaults. Apps mounted with
// MountApp start from these values and can override them per mount.
func (s *Server) SetDocument(doc Document) {
	s.docMu.Lock()
	defer s.docMu.Unlock()
	s.doc = doc
}

func (s *Server) document() Document {
	s.docMu.RLock()
	defer s.docMu.RUnlock()
	return s.doc
}

// Use appends middleware to the global chain. Unlike chi, it applies to the
// runtime, modules, and procedures as well as router routes, and it can be
// called after routes are registered.
func (s *Server) Use(middleware ...func(http.Handler) http.Handler) {
	if len(middleware) == 0 {
		return
	}
	s.chainMu.Lock()
	defer s.chainMu.Unlock()
	s.middleware = append(s.middleware, middleware...)
	s.chain = nil
}

// Router exposes the underlying chi router for scoped middleware, groups, and
// anything else the Server does not wrap. Middleware added through the router
// covers only routes registered through it, so prefer Use for auth and
// logging.
func (s *Server) Router() chi.Router { return s.mux }

// Get registers a GET route. Patterns are relative to the server scope.
func (s *Server) Get(pattern string, h http.HandlerFunc) {
	s.mux.Get(s.routePath(pattern), h)
}

// Post registers a POST route.
func (s *Server) Post(pattern string, h http.HandlerFunc) {
	s.mux.Post(s.routePath(pattern), h)
}

// Handle registers a handler for a pattern.
func (s *Server) Handle(pattern string, h http.Handler) {
	s.mux.Handle(s.routePath(pattern), h)
}

// MountApp mounts an application at pattern. It renders a full HTML document
// around the app view and also mounts a catch-all below the pattern so
// client-side paths and deep links reach the app. Use a request-scoped model
// with NewAppFunc to render the right state for a URL. Options override the
// server document defaults for this mount.
func (s *Server) MountApp(pattern string, app *App, opts ...AppOption) {
	doc := s.document()
	for _, opt := range opts {
		opt(&doc)
	}
	mountPath := s.routePath(pattern)
	handler := func(w http.ResponseWriter, r *http.Request) {
		req := r.WithContext(WithBasePath(r.Context(), s.basePath))
		view, err := app.view(req)
		if err != nil {
			s.logError(req, err)
			WriteError(w, req, Internal("internal server error", err))
			return
		}
		var page bytes.Buffer
		if err := WriteDocument(&page, req, doc, view); err != nil {
			s.logError(req, err)
			WriteError(w, req, Internal("internal server error", err))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page.Bytes())
	}
	s.mux.Get(mountPath, handler)
	if strings.Contains(mountPath, "*") {
		return
	}
	catch := strings.TrimSuffix(mountPath, "/") + "/*"
	if mountPath == "/" {
		catch = "/*"
	}
	s.mux.Get(catch, handler)
}

// RegisterProcedure registers a server procedure. Generated code calls this
// from a file-level registrar.
func (s *Server) RegisterProcedure(name string, fn ProcedureFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.procedures[name] = fn
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.serveHealth(w, r) {
		return
	}
	s.handler().ServeHTTP(w, r)
}

// handler builds and caches the middleware chain.
func (s *Server) handler() http.Handler {
	s.chainMu.Lock()
	defer s.chainMu.Unlock()
	if s.chain != nil {
		return s.chain
	}
	var h http.Handler = http.HandlerFunc(s.dispatch)
	for i := len(s.middleware) - 1; i >= 0; i-- {
		h = s.middleware[i](h)
	}
	s.chain = h
	return h
}

func (s *Server) dispatch(w http.ResponseWriter, r *http.Request) {
	if s.root == nil {
		if scope := s.matchScope(r.URL.Path); scope != nil {
			scope.handler().ServeHTTP(w, r)
			return
		}
	}
	s.dispatchLocal(w, r)
}

func (s *Server) dispatchLocal(w http.ResponseWriter, r *http.Request) {
	path := s.relativePath(r.URL.Path)
	switch {
	case path == RuntimePath:
		serveJS(w, runtimeclient.RuntimeJS)
	case strings.HasPrefix(path, ModulesPrefix):
		data, ok := lookupModule(path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		serveJS(w, data)
	case strings.HasPrefix(path, RPCPrefix):
		s.handleRPC(w, r, path)
	default:
		s.mux.ServeHTTP(w, r)
	}
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeErrorJSON(w, Errorf(http.StatusMethodNotAllowed, "method_not_allowed", "procedure calls require POST"))
		return
	}
	if !IsJSONContentType(r.Header.Get("Content-Type")) {
		writeErrorJSON(w, Errorf(http.StatusUnsupportedMediaType, "unsupported_media_type", "expected application/json"))
		return
	}
	if !s.sameOrigin(r) {
		writeErrorJSON(w, Forbidden("cross-origin procedure calls are not allowed"))
		return
	}
	name := strings.TrimPrefix(path, RPCPrefix)
	if name == "" || strings.Contains(name, "/") {
		writeErrorJSON(w, NotFound("unknown procedure"))
		return
	}
	s.mu.RLock()
	fn, ok := s.procedures[name]
	s.mu.RUnlock()
	if !ok {
		writeErrorJSON(w, NotFound("unknown procedure"))
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.maxBodyBytes))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeErrorJSON(w, Errorf(http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds %d bytes", s.maxBodyBytes))
			return
		}
		writeErrorJSON(w, BadRequest(err.Error()))
		return
	}
	result, err := fn(r.Context(), body)
	if err != nil {
		s.logError(r, err)
		writeErrorJSON(w, asError(err))
		return
	}
	_ = JSON(w, http.StatusOK, map[string]any{"result": result})
}

// sameOrigin rejects cross-site procedure calls. Requests without an Origin
// header, such as server-to-server and command-line clients, are allowed.
func (s *Server) sameOrigin(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "", "same-origin", "none":
	default:
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := r.Host
	if trustedProxy(r) || s.isTrustedProxy(r.RemoteAddr) {
		if forwarded := headerValue(r, "X-Forwarded-Host"); forwarded != "" {
			host = forwarded
		}
	}
	return strings.EqualFold(u.Host, host)
}

func serveJS(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}
