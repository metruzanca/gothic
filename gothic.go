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

// RPCService is a marker interface. Embed it in an interface to declare the
// methods that gothic should expose as client-callable procedures.
type RPCService interface{}

// UnmarshalArgs decodes the JSON argument array sent by the client runtime.
func UnmarshalArgs(body []byte, dst ...any) error {
	var payload struct {
		Args []json.RawMessage `json:"args"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return BadRequest(fmt.Sprintf("gothic: decode arguments: %v", err))
	}
	if len(payload.Args) != len(dst) {
		return BadRequest(fmt.Sprintf("gothic: got %d arguments, want %d", len(payload.Args), len(dst)))
	}
	for i := range dst {
		if err := json.Unmarshal(payload.Args[i], dst[i]); err != nil {
			return BadRequest(fmt.Sprintf("gothic: argument %d: %v", i, err))
		}
	}
	return nil
}

// Msg is an application message. Following Bubble Tea, it is any value and is
// usually inspected with a type switch.
type Msg = any

// Cmd is a command that performs work and returns a message.
type Cmd func() Msg

const commandOnly = "gothic: Batch and Sequence run in the browser"

// Batch runs commands concurrently in the browser.
//
// The Go stub exists so Init and Update compile. The generated client code
// recognizes the call and dispatches every non-nil result.
func Batch(cmds ...Cmd) Cmd {
	return func() Msg { panic(commandOnly) }
}

// Sequence runs commands in order in the browser.
//
// The Go stub exists so Init and Update compile. The generated client code
// recognizes the call and dispatches every non-nil result in order.
func Sequence(cmds ...Cmd) Cmd {
	return func() Msg { panic(commandOnly) }
}

// Children marks a props field that receives child content.
//
// Embed it in a props struct to let a component accept children, then render
// the children with { props.Children } in the view.
type Children struct{}

// PropsChanged is dispatched to a stateful component when its props change.
// Initial is true on mount, so Update can seed local state from props once.
type PropsChanged struct {
	Initial bool
}

// Model is a Bubble Tea-style application model.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Component
}

// App is a gothic application built from a Model.
type App struct {
	initial Model
	model   func(*http.Request) Model
}

// NewApp creates an application from its initial model.
func NewApp(initial Model) *App {
	return &App{initial: initial}
}

// NewAppFunc creates an application whose initial model depends on the
// request. Use it with MountApp to render deep links from the URL.
func NewAppFunc[M Model](model func(*http.Request) M) *App {
	return &App{model: func(r *http.Request) Model { return model(r) }}
}

func (a *App) view(r *http.Request) (Component, error) {
	model := a.initial
	if a.model != nil {
		model = a.model(r)
	}
	if model == nil {
		return nil, fmt.Errorf("gothic: app model is nil")
	}
	return model.View(), nil
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
