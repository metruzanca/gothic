<p align="center">
  <img src="assets/gothic-wordmark.svg" alt="Gothic" width="260">
</p>

Gothic is a batteries-included Go framework for building full-stack apps.
It prioritizes simplicity and productivity.

## Goals

- **Fullstack Go** Go is the only language you write
- **Simplicity** Optimized for readability and understanding whats going on
- **Opinionated** Fewer decisions to make, focus on building

## A taste

The syntax is under active development and subject to change. See the
[guestbook](examples/guestbook), [TodoMVC](examples/todo),
[accounts](examples/accounts), [Gothicbook](examples/gothicbook),
[router](examples/router), [nested components](examples/nested), and the
[virtualized list](virtual/example) for working code.
```templ
// app.gth
package main

import (
	"context"

	"github.com/metruzanca/gothic"
)

type Message struct{ Text string }

type EntryProps struct{ Item Message }

type Model struct {
	Draft    string
	Messages []Message
}

type DraftChanged struct{ Text string }
type PostClicked struct{}
type Posted struct{ Message Message }
type PostFailed struct{ Message string }

// Procedures declares the RPC calls available to the client.
// The embedded gothic.RPCService marks it for code generation.
type Procedures interface {
	gothic.RPCService
	Post(ctx context.Context, text string) (Message, error)
}

func (m Model) Init() gothic.Cmd { return nil }

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case DraftChanged:
		m.Draft = msg.Text
	case PostClicked:
		text := m.Draft
		if text == "" {
			return m, nil
		}
		m.Draft = ""
		return m, func() gothic.Msg {
			posted, err := Post(text)
			if err != nil {
				return PostFailed{Message: err.Error()}
			}
			return Posted{Message: posted}
		}
	case Posted:
		m.Messages = append(m.Messages, msg.Message)
	case PostFailed:
		m.Messages = append(m.Messages, Message{Text: msg.Message})
	}
	return m, nil
}

view Entry(props EntryProps) {
	<li>{ props.Item.Text }</li>
}

view App(model Model) {
	<div class="app">
		<input value={ model.Draft } oninput={ DraftChanged{Text: event.target.value} } onenter={ PostClicked{} } />
		<button onclick={ PostClicked{} }>Post</button>
		if len(model.Messages) == 0 {
			<p>No messages yet.</p>
		} else {
			<ul>
				for i, message := range model.Messages {
					<Entry key={ i } item={ message } />
				}
			</ul>
		}
	</div>
}
```

```go
// main.go
package main

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/metruzanca/gothic"
)

type procedures struct{}

func (procedures) Post(ctx context.Context, text string) (Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Message{}, errors.New("write something first")
	}
	return Message{Text: text}, nil
}

func main() {
	s := gothic.NewServer()
	RegisterProcedures(s, procedures{})
	s.MountApp("/", gothic.NewApp(Model{}))
	log.Fatal(s.ListenAndServe(":8080"))
}
```

The full example adds an in-memory store so messages survive reloads, plus
tests: [examples/guestbook](examples/guestbook).

## UI kit

The optional UI kit is a separately versioned Go module:

```sh
go get github.com/metruzanca/gothic/ui
```

See [ui/README.md](ui/README.md) and the [Gothicbook demo](examples/gothicbook).

## Status

Early, working proof of concept. Every example above runs end to end: server
rendering, hydration, the update loop, typed procedures, pure and stateful
components, conditionals, keyed lists, routing, an installable UI kit, and
server-rendered documents with OpenGraph metadata. Browser interop ships as the
separate `browser` module, long-list windowing as the separate `virtual` module,
and sessions and auth as the separate `session` and `auth` modules. The
[guide](docs/guide/01-quickstart.md) covers the whole loop, and
[SYNTAX.md](docs/SYNTAX.md) is the full reference.

## Credits

The gothic logo mark and wordmark are by [Mackenzie Child](https://mackenziechild.me)
([X](https://x.com/mackenziechild)). See [CREDITS.md](CREDITS.md).

## Learn more

Start with the [quickstart guide](docs/guide/01-quickstart.md).

Guide:

- [State and commands](docs/guide/02-state.md)
- [Components](docs/guide/03-components.md)
- [Routing and documents](docs/guide/04-routing.md)
- [Browser interop](docs/guide/05-browser.md)
- [Long lists](docs/guide/06-long-lists.md)
- [Architecture](docs/guide/07-architecture.md)

Reference:

- [Modules](docs/MODULES.md)
- [Syntax](docs/SYNTAX.md)
- [Roadmap](docs/PLAN.md)
- [Pitch](docs/PITCH.md)
- [Releasing](docs/RELEASING.md)
