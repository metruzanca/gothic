//go:generate go run github.com/metruzanca/gothic/cmd/gothic generate .

package main

import (
	"bytes"
	"context"
	_ "embed"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/ui"
)

//go:embed style.css
var pageCSS []byte

//go:embed assets/gothic-mark.svg
var pageMark []byte

//go:embed assets/gothic-wordmark.svg
var pageWordmark []byte

type counterStore struct {
	count atomic.Int64
}

func (c *counterStore) increment() int {
	return int(c.count.Add(1))
}

type procedures struct {
	counter *counterStore
}

func (p procedures) Increment(ctx context.Context) (int, error) {
	return p.counter.increment(), nil
}

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	log.Fatal(s.ListenAndServe(addr))
}

func mountApp(s *gothic.Server, counter *counterStore) {
	RegisterProcedures(s, procedures{counter: counter})
	ui.MountStyles(s)
	s.Get("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(pageCSS)
	})
	s.Get("/gothic-mark.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write(pageMark)
	})
	s.Get("/gothic-wordmark.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write(pageWordmark)
	})
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		var page bytes.Buffer
		page.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Gothicbook</title><link rel="stylesheet" href="/gothic-ui.css"><link rel="stylesheet" href="/style.css"></head><body>`)
		if err := gothic.NewApp(NewModel(int(counter.count.Load()))).RenderTo(r.Context(), &page); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		page.WriteString(`</body></html>`)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page.Bytes())
	})
}
