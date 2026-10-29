// Package gothic provides the server and component runtime for gothic.
package gothic

import (
	"context"
	"encoding/json"
	"fmt"
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

// RPCService is a marker interface. Embed it in an interface to declare the
// methods that gothic should expose as client-callable procedures.
type RPCService interface{}

// UnmarshalArgs decodes the JSON argument array sent by the client runtime.
func UnmarshalArgs(body []byte, dst ...any) error {
	var payload struct {
		Args []json.RawMessage `json:"args"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("gothic: decode arguments: %w", err)
	}
	if len(payload.Args) != len(dst) {
		return fmt.Errorf("gothic: got %d arguments, want %d", len(payload.Args), len(dst))
	}
	for i := range dst {
		if err := json.Unmarshal(payload.Args[i], dst[i]); err != nil {
			return fmt.Errorf("gothic: argument %d: %w", i, err)
		}
	}
	return nil
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

// RenderTo writes the application's initial view to w.
func (a *App) RenderTo(ctx context.Context, w io.Writer) error {
	return a.initial.View().Render(ctx, w)
}

// Render writes the application's initial view to the response.
func (a *App) Render(w http.ResponseWriter, r *http.Request) error {
	return a.RenderTo(r.Context(), w)
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
