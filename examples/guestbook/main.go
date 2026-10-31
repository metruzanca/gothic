//go:generate go run ../../cmd/gothic generate .

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/metruzanca/gothic"
)

// guestbook keeps every message in memory. All visitors share the list, and it
// resets when the process restarts.
type guestbook struct {
	mu       sync.Mutex
	messages []Message
}

func (g *guestbook) Messages() []Message {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]Message(nil), g.messages...)
}

func (g *guestbook) Post(ctx context.Context, text string) (Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Message{}, errors.New("write something first")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	msg := Message{Text: text}
	g.messages = append(g.messages, msg)
	return msg, nil
}

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	g := &guestbook{}
	s := gothic.NewServer()
	RegisterProcedures(s, g)
	s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
		return NewModel(g.Messages())
	}))
	log.Fatal(s.ListenAndServe(addr))
}
