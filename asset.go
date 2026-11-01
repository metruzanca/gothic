package gothic

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"mime"
	"net/http"
	"path"
	"time"
)

const defaultAssetCacheControl = "public, max-age=3600"

// Asset serves one in-memory asset at pattern. It sets the content type from
// the file extension, sends an ETag, and handles conditional and range
// requests for GET and HEAD.
func (s *Server) Asset(pattern string, data []byte) {
	contentType := mime.TypeByExtension(path.Ext(pattern))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	etag := `"` + assetHash(data) + `"`
	serve := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", defaultAssetCacheControl)
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

func assetHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}
