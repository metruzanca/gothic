//go:generate go run ../../cmd/gothic generate .

package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/metruzanca/gothic"
)

type procedures struct{}

func (procedures) Ping(ctx context.Context) (string, error) {
	if os.Getenv("PING_FAIL") != "" {
		return "", errors.New("ping failed")
	}
	return "pong", nil
}

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	app := gothic.NewApp(NewModel())
	s := gothic.NewServer()
	RegisterView(s, procedures{})
	s.MountApp("/", app)
	log.Fatal(s.ListenAndServe(addr))
}
