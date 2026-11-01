package gothic

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// DevToolsWorkspacePath is the path Chrome DevTools requests to discover a
// local workspace.
const DevToolsWorkspacePath = "/.well-known/appspecific/com.chrome.devtools.json"

// WithDevMode enables development-only endpoints and records the project
// directory. An empty directory uses the current working directory. The
// DevTools workspace response is served only in dev mode because it exposes an
// absolute filesystem path.
func WithDevMode(projectDir string) Option {
	return func(s *Server) {
		s.devMode = true
		if projectDir == "" {
			projectDir, _ = os.Getwd()
		}
		if absolute, err := filepath.Abs(projectDir); err == nil {
			projectDir = absolute
		}
		s.projectDir = filepath.Clean(projectDir)
	}
}

// serveDevTools handles Chrome DevTools workspace discovery before the
// middleware chain. It returns JSON instead of falling through to the app
// catch-all, so the probe never receives HTML.
func (s *Server) serveDevTools(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != DevToolsWorkspacePath {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeSpecialJSON(w, r, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return true
	}
	if !s.devMode || s.projectDir == "" {
		writeSpecialJSON(w, r, http.StatusNotFound, map[string]string{"error": "not found"})
		return true
	}
	payload := map[string]any{
		"workspace": map[string]string{
			"root": s.projectDir,
			"uuid": workspaceUUID(s.projectDir),
		},
	}
	writeSpecialJSON(w, r, http.StatusOK, payload)
	return true
}

// workspaceUUID derives a stable UUID from the project directory. DevTools
// uses it to recognize the same workspace across restarts.
func workspaceUUID(projectDir string) string {
	sum := sha256.Sum256([]byte("gothic-workspace:" + projectDir))
	b := sum[:16]
	b[6] = (b[6] & 0x0f) | 0x50 // version 5 layout
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
