//go:generate go run ../../cmd/gothic generate .

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/examples/pingpong/views"
)

type procedures struct{}

func (procedures) Ping(ctx context.Context) (string, error) {
	return "pong", nil
}

func newServer() *gothic.Server {
	s := gothic.NewServer()
	views.RegisterPingPong(s, procedures{})
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := gothic.Render(w, r, views.PingPong()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	return s
}

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	log.Fatal(newServer().ListenAndServe(addr))
}
