# Iteration 13: props structs for components

Status: complete. Components declare a props struct, named tag props map to
exported fields, pointer fields are optional, and `Init` receives props.

## Goal

Replace long component prop lists with one props struct without changing call
sites.

```gothic
<ui.Tabs items={ model.Tabs } active={ model.ActiveTab } />
```

The compiler maps `items` to `Props.Items` and `active` to `Props.Active`,
then builds the props struct for Go and a plain object for JavaScript.

## Stateful components

```go
type TabsProps struct {
	Items  []Tab
	Active *int
}

type TabsModel struct {
	Active int
}

func (m TabsModel) Init(props TabsProps) gothic.Cmd { return nil }

func (m TabsModel) Update(msg gothic.Msg, props TabsProps) (TabsModel, gothic.Cmd) {
	switch msg := msg.(type) {
	case TabClicked:
		m.Active = msg.Index
	case gothic.PropsChanged:
	}
	if m.Active >= len(props.Items) {
		m.Active = len(props.Items) - 1
	}
	return m, nil
}
```

```gothic
view Tabs(model TabsModel, props TabsProps) {
	<div>
		for i, item := range props.Items {
			<button onclick={ TabClicked{Index: i} }>{ item.Label }</button>
		}
	</div>
}
```

- The props parameter must be named `props`.
- The struct lives in the component prelude.
- Stateful components with props require the struct.
- Components with no props keep `view Box(model BoxModel)`, `Init()`, and
  `Update(msg)`.

## Field mapping

- Tag names map to exported fields case-insensitively.
- `items` maps to `Items`, `activeTab` maps to `ActiveTab`.
- Boolean shorthand works: `primary` sets `Primary: true`.
- `key` and `route` remain reserved.
- Two fields that differ only by case are rejected as ambiguous.
- A field named `Key` is rejected because `key` is a list key.

Non-pointer fields are required. Pointer fields are optional.

## Optional pointer props

```go
type TabsProps struct {
	Items  []Tab
	Active *int
}
```

Omit `active` and `props.Active` is nil in Go and `null` in the client props
object. The client object always includes optional fields as `null`, so nil
checks are consistent.

```gothic
if props.Active != nil {
	<p>{ *props.Active }</p>
}
```

- `*props.Active` compiles to `props.Active` on the client.
- `props.Active != nil` compiles to `props.Active != null`.
- Inline `&model.Field` is rejected. Store a pointer-valued field in the model
  or pass `nil`.

## Pure components

Pure components can use a props struct too:

```go
type CardProps struct {
	Title string
	Body  *string
}

view Card(props CardProps) {
	<section>
		<h2>{ props.Title }</h2>
		if props.Body != nil {
			<p>{ *props.Body }</p>
		}
	</section>
}
```

Call sites stay unchanged:

```gothic
<ui.Card title="Hello" />
```

During expansion, `props` becomes a struct literal. The Go renderer sees a
struct literal, and the JavaScript compiler emits a plain object.

Individual pure props remain supported during migration:

```gothic
view Badge(text string) { ... }
```

New pure components should prefer a props struct.

## External pure components

The UI kit defines props structs in its own package. The CLI:

- reads the component package preludes,
- extracts props structs,
- qualifies the type for generated Go, such as `ui.ButtonProps`,
- adds the qualified type to the expansion table.

Library Go generation now includes preludes from all component files, so
stateful views that call pure components can reference shared props types.

Apps using external pure component props types must import the component
package in the app prelude, as they already do for stateful components and
external model types.

## Props in Init

`Init` can take the props struct:

```go
func (m TabsModel) Init(props TabsProps) gothic.Cmd
```

The generated component receives `props` at mount and runs `init(props)`.
`Init` can use props for commands and synchronous `js` effects.

`Init` still cannot seed the model. Component state starts at the zero value.
A prop-aware model constructor is a separate change.

## Manifest

External stateful components now record:

```json
{
  "props_type": "TabsProps",
  "props": [
    {"name": "Items", "type": "[]Tab"}
  ]
}
```

Field names are exported Go names. Types drive optional pointer detection.

## UI kit and examples

- UI kit stateful components use props structs: Tabs, Accordion, Counter,
  Disclosure.
- UI kit pure components use props structs: Button, Card, Field.
- Gothicbook `Code`, components `Status` and `HistoryItem`, and TodoMVC
  `TodoItem` use props structs.
- Call sites remain named tag props.

## Commits

- `5f5b050` stateful props structs and named tag mapping
- `a97ac2d` optional pointer props
- `1fa9a13` local pure props structs
- `c2d6559` external pure props structs and UI kit migration
- `4107238` props in component `Init`

## Verification

Core:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
```

Browser checks:

- Tabs clamps its active index after removing a tab,
- optional pointer props render when provided and stay absent when omitted,
- Field slots and Counter inside a slot still work,
- component `Init` receives props and runs its command,
- no console errors.

## Limits

- `Init` cannot seed component state from props yet,
- non-pointer props are required,
- address-of expressions are not supported in views,
- prop change detection is reference-based,
- pure components still accept individual props during migration,
- controlled components and function-valued props remain unsupported,
- component-declared procedures remain unsupported and procedures stay
  App-owned.

## Follow-ups

- Prop-seeded initial component state.
- Nested stateful components.
- Path parameters and route matching helpers.
- The virtualized list example.
