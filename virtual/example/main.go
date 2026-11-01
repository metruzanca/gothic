//go:generate go run ../../cmd/gothic generate .

package main

import (
	_ "embed"
	"log"
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
	log.Fatal(newServer().ListenAndServe(addr))
}

func newServer() *gothic.Server {
	s := gothic.NewServer()
	s.Asset("/style.css", pageCSS)
	s.MountApp("/", gothic.NewApp(NewModel()),
		gothic.WithDocument(gothic.Document{
			Title:  "Virtualized list",
			Styles: []string{"/style.css"},
		}),
	)
	return s
}
