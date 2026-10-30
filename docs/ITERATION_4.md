# Iteration 4: stateful components and the UI kit

Status: working proof of concept. This iteration adds self-contained stateful
components, an installable UI kit, and the Gothicbook design system example.

## Goal

Prove that components can have their own state and update loop while still being
used from ordinary gothic views:

- a stateful component owns its model, `Init`, `Update`, and view,
- the parent passes props but does not read or write child state,
- components work in static positions and keyed lists,
- components can be shipped in a separately versioned package,
- a small UI kit can be installed and used with namespaced tags.

## Repository modules

The repository now contains:

```
github.com/metruzanca/gothic          core framework
github.com/metruzanca/gothic/ui       optional precompiled UI kit
examples/gothicbook                   example module
```

The UI kit and Gothicbook have their own `go.mod` files. Gothicbook uses local
`replace` directives so it builds against the checkout during development.

## Stateful component model

A stateful component is a view whose first parameter is a model with `Init` and
`Update` methods:

```go
type TabsModel struct{ Active int }

func (m TabsModel) Init() gothic.Cmd { return nil }

func (m TabsModel) Update(msg gothic.Msg) (TabsModel, gothic.Cmd) {
	switch msg := msg.(type) {
	case TabClicked:
		m.Active = msg.Index
	}
	return m, nil
}
```

```gothic
view Tabs(model TabsModel, items []Tab) {
	<div class="ui-tabs">
		for i, item := range items {
			<button key={ i } class:active={ model.Active == i } onclick={ TabClicked{Index: i} }>
				{ item.Label }
			</button>
		}
	</div>
}
```

Each instance gets a wrapper with its own serialized state:

```html
<div data-gothic-component="Tabs" data-gothic-component-id="c3">
	<script type="application/json" data-gothic-component-state>{"Active":0}</script>
	...
</div>
```

The app mounts each wrapper with a generated component module. Parent renders
call `updateProps`, but child state is preserved. App-level bindings skip
component wrappers so marker IDs cannot collide.

## Component limits

Supported in this iteration:

- self-contained state,
- static positions,
- keyed lists,
- props updates from the parent,
- component-local event dispatch,
- app-local components,
- precompiled external components.

Not supported yet:

- commands or procedures inside components,
- function-valued callback props and controlled components,
- props inside component Init/Update,
- stateful components nested inside stateful components,
- creating list instances for components whose views contain loops,
- stateful components nested inside another list item element,
- component children and slots.

## UI kit package

The UI kit lives in [`ui`](../ui) as a separate Go module:

```sh
go get github.com/metruzanca/gothic/ui
```

It contains five proof-of-concept components.

Pure components: `Button`, `Card`.

Stateful components: `Tabs`, `Accordion`, `Counter`.

`Counter` uses the pure `Button` component internally.

Tabs owns the active tab, Accordion owns the open item, and Counter owns its
count. The pure components are expanded into the consumer app.

Add the package to an app with a directive:

```go
//gothic:components github.com/metruzanca/gothic/ui
```

Then use namespaced tags:

```gothic
<ui.Button label="Save" primary={ true } onclick={ SaveClicked{} } />
<ui.Card title="Title" body="Body" />
<ui.Tabs items={ model.Tabs } />
<ui.Accordion items={ model.Accordion } />
<ui.Counter label="Increment on client" />
```

The CLI resolves the package with `go list`, reads
`gothic-components.json`, imports the generated Go wrappers, and imports the
generated JavaScript module. The UI package serves its CSS through
`ui.MountStyles(s)`.

## Gothicbook

[`examples/gothicbook`](../examples/gothicbook) is a one-page design system demo.
It shows every UI kit component, its live behavior, and a short usage snippet.
It has no database, auth, routing, or playground.

## Verification

Core:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
```

UI module:

```sh
cd ui
go test ./...
```

Gothicbook:

```sh
cd examples/gothicbook
go generate
go test ./...
```

Browser checks:

- tabs switch panels independently,
- accordion items open and close,
- code snippets are not overwritten by component text updates,
- component wrappers keep their own state.
