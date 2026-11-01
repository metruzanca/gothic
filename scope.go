package gothic

import (
	"context"
	"strings"
)

type basePathKey struct{}

// WithBasePath returns a context that carries the mount base path. MountApp
// uses it so generated code can resolve framework URLs under a scope.
func WithBasePath(ctx context.Context, base string) context.Context {
	if base == "" {
		return ctx
	}
	return context.WithValue(ctx, basePathKey{}, base)
}

// BasePath returns the mount base path from the context. It is empty for apps
// mounted at the root.
func BasePath(ctx context.Context) string {
	base, _ := ctx.Value(basePathKey{}).(string)
	return base
}

// Scope creates a child server mounted below prefix. The child shares the
// router, the module registry, logger, and transport settings, and owns its
// procedures, document defaults, and middleware.
//
//	s := gothic.NewServer()
//	admin := s.Scope("/admin")
//	RegisterAdminProcedures(admin, impl)
//	admin.MountApp("/", adminApp)
//
// Generated registrars keep working because a scoped server is still a
// *gothic.Server.
func (s *Server) Scope(prefix string) *Server {
	base := normalizeScopePrefix(prefix)
	if base == "" {
		panic("gothic: scope prefix must not be the root")
	}
	root := s.rootServer()
	child := &Server{
		mux:                  root.mux,
		procedures:           map[string]ProcedureFunc{},
		doc:                  s.document(),
		logger:               s.logger,
		addr:                 s.addr,
		maxBodyBytes:         s.maxBodyBytes,
		readTimeout:          s.readTimeout,
		readHeaderTimeout:    s.readHeaderTimeout,
		writeTimeout:         s.writeTimeout,
		idleTimeout:          s.idleTimeout,
		shutdownTimeout:      s.shutdownTimeout,
		requestTimeout:       0,
		trustedProxies:       s.trustedProxies,
		useDefaultMiddleware: false,
		root:                 root,
		basePath:             joinBasePath(s.basePath, base),
	}
	root.registerScope(child)
	return child
}

// normalizeScopePrefix returns a leading-slash prefix without a trailing
// slash. The root prefix returns an empty string.
func normalizeScopePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || prefix == "/" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimRight(prefix, "/")
}

func joinBasePath(parent, child string) string {
	if parent == "" {
		return child
	}
	if child == "" {
		return parent
	}
	return parent + child
}

func (s *Server) rootServer() *Server {
	if s.root != nil {
		return s.root
	}
	return s
}

func (s *Server) registerScope(child *Server) {
	root := s.rootServer()
	root.scopesMu.Lock()
	root.scopes = append(root.scopes, child)
	root.scopesMu.Unlock()
}

// matchScope returns the longest scope whose path prefix contains path.
func (s *Server) matchScope(path string) *Server {
	s.scopesMu.RLock()
	defer s.scopesMu.RUnlock()
	var best *Server
	for _, scope := range s.scopes {
		if !hasPathPrefix(path, scope.basePath) {
			continue
		}
		if best == nil || len(scope.basePath) > len(best.basePath) {
			best = scope
		}
	}
	return best
}

func hasPathPrefix(path, base string) bool {
	if base == "" {
		return false
	}
	if path == base {
		return true
	}
	return strings.HasPrefix(path, base+"/")
}

// relativePath strips the scope prefix from a request path.
func (s *Server) relativePath(path string) string {
	if s.basePath == "" {
		return path
	}
	if path == s.basePath {
		return "/"
	}
	trimmed := strings.TrimPrefix(path, s.basePath)
	if trimmed == "" {
		return "/"
	}
	return trimmed
}

// routePath applies the scope prefix to a route pattern.
func (s *Server) routePath(pattern string) string {
	if s.basePath == "" {
		return pattern
	}
	if pattern == "" || pattern == "/" {
		return s.basePath
	}
	if strings.HasPrefix(pattern, "/") {
		return s.basePath + pattern
	}
	return s.basePath + "/" + pattern
}
