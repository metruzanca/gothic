//go:generate go run ../../cmd/gothic generate .

package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"

	"github.com/metruzanca/gothic"
)

//go:embed style.css
var pageCSS []byte

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	s := gothic.NewServer()
	mountApp(s)
	log.Fatal(s.ListenAndServe(addr))
}

func mountApp(s *gothic.Server) {
	s.Get("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(pageCSS)
	})
	s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
		return NewModel(r.URL.Path)
	}))
}
