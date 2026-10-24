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
