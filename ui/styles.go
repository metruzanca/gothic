package ui

import (
	_ "embed"
	"net/http"

	"github.com/metruzanca/gothic"
)

//go:embed ui.css
var Styles []byte

// MountStyles serves the UI kit stylesheet at /gothic-ui.css.
func MountStyles(s *gothic.Server) {
	s.Get("/gothic-ui.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(Styles)
	})
}
