//go:generate go run ../../cmd/gothic generate .

package main

import (
	"log"
	"os"

	"github.com/metruzanca/gothic"
)

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	s := gothic.NewServer()
	s.MountApp("/", gothic.NewApp(NewModel()),
		gothic.WithDocument(gothic.Document{
			Title:       "Nested stateful components",
			Description: "Nested stateful components with local and shared state in gothic.",
		}),
	)
	log.Fatal(s.ListenAndServe(addr))
}
