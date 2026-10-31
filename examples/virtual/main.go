//go:generate go run ../../cmd/gothic generate .

package main

import (
	"bytes"
	_ "embed"
	"log"
	"net/http"
	"os"

	"github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/ui"
)

//go:embed style.css
var pageCSS []byte

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	log.Fatal(newServer().ListenAndServe(addr))
}

func newServer() *gothic.Server {
	s := gothic.NewServer()
	ui.MountStyles(s)
	s.Get("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(pageCSS)
	})
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		var page bytes.Buffer
		page.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Virtualized list</title><link rel="stylesheet" href="/gothic-ui.css"><link rel="stylesheet" href="/style.css"></head><body>`)
		if err := gothic.NewApp(NewModel()).RenderTo(r.Context(), &page); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		page.WriteString(`</body></html>`)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page.Bytes())
	})
	return s
}
