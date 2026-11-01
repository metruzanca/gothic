package gothic

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const defaultAssetCacheControl = "public, max-age=3600"

// StaticOption customizes how a static asset is served.
type StaticOption func(*staticConfig)

type staticConfig struct {
	cacheControl string
}

func newStaticConfig(opts []StaticOption) staticConfig {
	cfg := staticConfig{cacheControl: defaultAssetCacheControl}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithCacheControl sets the Cache-Control header for a static asset.
func WithCacheControl(value string) StaticOption {
	return func(cfg *staticConfig) {
		if value != "" {
			cfg.cacheControl = value
		}
	}
}

// WithImmutableAssets marks an asset as content-addressed, so it can be cached
// for a year. Pair it with a hashed file name or query string.
func WithImmutableAssets() StaticOption {
	return WithCacheControl("public, max-age=31536000, immutable")
}

// Asset serves one in-memory asset at pattern. It sets the content type from
// the file extension, sends an ETag, and handles conditional and range
// requests for GET and HEAD.
func (s *Server) Asset(pattern string, data []byte, opts ...StaticOption) {
	cfg := newStaticConfig(opts)
	contentType := mime.TypeByExtension(path.Ext(pattern))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	etag := `"` + assetHash(data) + `"`
	serve := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", cfg.cacheControl)
		http.ServeContent(w, r, path.Base(pattern), time.Time{}, bytes.NewReader(data))
	}
	s.Handle(pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			WriteError(w, r, Errorf(http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed"))
			return
		}
		serve(w, r)
	}))
}

// Static serves fsys below pattern. The pattern is a URL prefix such as
// "/assets", and the filesystem root maps to that prefix. Requests below the
// prefix are served with Last-Modified and range support.
func (s *Server) Static(pattern string, fsys fs.FS, opts ...StaticOption) {
	if fsys == nil {
		panic("gothic: Static requires a filesystem")
	}
	cfg := newStaticConfig(opts)
	prefix := strings.TrimRight(pattern, "/")
	if prefix == "" {
		prefix = "/"
	}
	fullPrefix := s.routePath(prefix)
	files := http.StripPrefix(fullPrefix, http.FileServerFS(fsys))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			WriteError(w, r, Errorf(http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed"))
			return
		}
		w.Header().Set("Cache-Control", cfg.cacheControl)
		files.ServeHTTP(w, r)
	})
	if fullPrefix != "/" {
		s.mux.Handle(fullPrefix, http.RedirectHandler(fullPrefix+"/", http.StatusMovedPermanently))
		s.mux.Handle(fullPrefix+"/*", handler)
		return
	}
	s.mux.Handle("/*", handler)
}

// StaticDir serves files from dir below pattern.
func (s *Server) StaticDir(pattern, dir string, opts ...StaticOption) {
	s.Static(pattern, os.DirFS(dir), opts...)
}

func assetHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}
