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
[Gothicbook](examples/gothicbook), [router](examples/router),
[nested components](examples/nested), and
[virtualized list](examples/virtual) examples for working code.
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
It includes a `VirtualList` component for fixed-height virtualized lists.

## Status

Early, working proof of concept. Every example above runs end to end: server
rendering, hydration, the update loop, typed procedures, pure and stateful
components, conditionals, keyed lists, routing, an installable UI kit, and
server-rendered documents with OpenGraph metadata. Current syntax and limits
live in [SYNTAX.md](docs/SYNTAX.md) and [PLAN.md](docs/PLAN.md).

## Credits

The gothic logo mark and wordmark are by [Mackenzie Child](https://mackenziechild.me)
([X](https://x.com/mackenziechild)). See [CREDITS.md](CREDITS.md).

## Learn more

- [Syntax](docs/SYNTAX.md)
- [Roadmap](docs/PLAN.md)
- [Proof of concept](docs/ITERATION_1.md)
- [Components](docs/ITERATION_2.md)
- [TodoMVC on SQLite](docs/ITERATION_3.md)
- [Stateful components and UI kit](docs/ITERATION_4.md)
- [App-owned actions](docs/ITERATION_5.md)
- [Conditional rendering](docs/ITERATION_6.md)
- [Stateful branches and lists](docs/ITERATION_7.md)
- [Component children and slots](docs/ITERATION_8.md)
- [Browser interop and URL state](docs/ITERATION_9.md)
- [Actionable diagnostics](docs/ITERATION_10.md)
- [Component props and Init](docs/ITERATION_11.md)
- [Component commands and destroy](docs/ITERATION_12.md)
- [Component props structs](docs/ITERATION_13.md)
- [Prop-seeded component state](docs/ITERATION_14.md)
- [Broader client Go subset](docs/ITERATION_15.md)
- [Path parameters](docs/ITERATION_16.md)
- [Nested stateful components](docs/ITERATION_17.md)
- [Virtualized list](docs/ITERATION_18.md)
- [Virtual list package and component](docs/ITERATION_19.md)
- [App-defined row views](docs/ITERATION_20.md)
- [Documents and social previews](docs/ITERATION_21.md)
- [Pitch](docs/PITCH.md)

Iteration notes are historical records of the state at that time. Current
syntax and limits live in [SYNTAX.md](docs/SYNTAX.md) and
[PLAN.md](docs/PLAN.md).
