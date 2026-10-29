// Package gothic provides the server and component runtime for gothic.
package gothic

import (
	"context"
	"io"
	"net/http"
	"sync"
)

// Component renders HTML into w.
type Component interface {
	Render(ctx context.Context, w io.Writer) error
}

// ComponentFunc adapts a function to Component.
type ComponentFunc func(ctx context.Context, w io.Writer) error

// Render implements Component.
func (f ComponentFunc) Render(ctx context.Context, w io.Writer) error {
	return f(ctx, w)
}

// Render writes c to the response.
func Render(w http.ResponseWriter, r *http.Request, c Component) error {
	return c.Render(r.Context(), w)
}

// Msg is an application message. Following Bubble Tea, it is any value and is
// usually inspected with a type switch.
type Msg = any

// Cmd is a command that performs work and returns a message.
type Cmd func() Msg

// Model is a Bubble Tea-style application model.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Component
}

// App is a gothic application built from a Model.
type App struct {
	initial Model
}

// NewApp creates an application from its initial model.
func NewApp(initial Model) *App {
	return &App{initial: initial}
}

// Render writes the application's initial view to the response.
func (a *App) Render(w http.ResponseWriter, r *http.Request) error {
	return a.initial.View().Render(r.Context(), w)
}

var (
	moduleMu sync.RWMutex
	modules  = map[string][]byte{}
)

// RegisterModule registers a generated JavaScript module to be served by Server.
// Generated code calls this from init.
func RegisterModule(path string, data []byte) {
	moduleMu.Lock()
	defer moduleMu.Unlock()
	modules[path] = data
}

func lookupModule(path string) ([]byte, bool) {
	moduleMu.RLock()
	defer moduleMu.RUnlock()
	data, ok := modules[path]
	return data, ok
}
