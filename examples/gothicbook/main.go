//go:generate go run github.com/metruzanca/gothic/cmd/gothic generate .

package main

import (
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

//go:embed assets/og.png
var pageOG []byte

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
	serveFile(s, "/style.css", "text/css; charset=utf-8", pageCSS)
	serveFile(s, "/gothic-mark.svg", "image/svg+xml", pageMark)
	serveFile(s, "/gothic-wordmark.svg", "image/svg+xml", pageWordmark)
	serveFile(s, "/og.png", "image/png", pageOG)
	s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
		return NewModel(int(counter.count.Load()))
	}),
		gothic.WithTitle("Gothicbook"),
		gothic.WithSiteName("Gothic"),
		gothic.WithDescription("A tiny design system demo for the installable gothic UI kit."),
		gothic.WithImage("/og.png"),
		gothic.WithStyles("/gothic-ui.css", "/style.css"),
	)
}

func serveFile(s *gothic.Server, pattern, contentType string, data []byte) {
	s.Get(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	})
}
