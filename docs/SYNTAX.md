# gothic syntax

This is the reference for the gothic view language and the Go surface around it.
For a guided tour, start with the [quickstart](guide/01-quickstart.md).

A `.gth` file contains ordinary Go followed by one or more views. One view,
`App`, is the root. The others are components. `main.go` remains the entrypoint.

## File layout

A `.gth` app file has two sections:

1. A Go prelude: package, imports, model, messages, `Init`, `Update`, commands,
   and procedure interfaces.
2. One or more views, starting with a line that begins with `view`.

Everything before the first top-level `view` line is parsed as Go. Everything
from that line onward is parsed as gothic markup.

```templ
package app

import "github.com/metruzanca/gothic"

type Model struct{ Count int }

type Increment struct{}

func (m Model) Init() gothic.Cmd { return nil }

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case Increment:
		m.Count++
	}
	return m, nil
}

view App(model Model) {
	<div>{ model.Count }</div>
}
```

A component file contains only a package clause and views. It has no model and
no `Init` or `Update`. Declare a separate component's props struct in the app
prelude. Pure components may live in the app file or a separate `.gth` file in
the same package. Stateful components live in the app file.

## Application model

The app follows Bubble Tea and the Elm Architecture. The model implements:

```go
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Component
}
```

- `Msg` is `any`.
- `Cmd` is `func() Msg`.
- `Update` returns the next model and an optional command.
- `Init` returns an optional command.
- `View` is generated from the `App` view.

`main.go` mounts the app:

```go
s := gothic.NewServer()
RegisterProcedures(s, procedures{})
s.MountApp("/", gothic.NewApp(Model{}))
s.ListenAndServe(":8080")
```

## Views

### App view

```templ
view App(model Model) {
	<div class="app">...</div>
}
```

The App view takes one model parameter. Its name and type are used to generate
the Go `View` method and the client renderer.

### Pure components

A pure component takes exactly one parameter named `props`, backed by a props
struct:

```templ
type BadgeProps struct {
	Text   string
	Hidden bool
}

view Badge(props BadgeProps) {
	<span class="badge" class:hidden={ props.Hidden }>{ props.Text }</span>
}
```

Call it with a capitalized tag:

```templ
<Badge text={ model.Status } hidden />
```

- Tag names map to fields case-insensitively.
- A tag without a value passes `true`.
- Non-pointer fields are required.
- Pointer fields are optional. Omitting the tag leaves the field `nil`.

Pure components are stateless. They render exactly one root element and should
reference only their props and package-level declarations. The compiler expands
them into the caller before generating Go and JavaScript.

### Children

A pure component accepts children when its props struct embeds `gothic.Children`:

```templ
type FieldProps struct {
	gothic.Children
	Label string
}

view Field(props FieldProps) {
	<div class="field">
		<span class="field-label">{ props.Label }</span>
		{ props.Children }
	</div>
}
```

```templ
<Field label="Your name">
	<input value={ model.Name } oninput={ NameChanged{Text: event.target.value} } />
</Field>
```

- Children are expanded in the caller's scope, so they can use the App model,
  loop variables, events, conditionals, and stateful components.
- The component must render `{ props.Children }` exactly once.
- A named `Children gothic.Children` field works too.
- Children are available on pure components only.

### Stateful components

A stateful component's first parameter is a model with `Init` and `Update`. A
second parameter is an optional props struct:

```go
type TabsProps struct {
	Items []Tab
}

type TabsModel struct {
	Active int
}

func (m TabsModel) Init() gothic.Cmd { return nil }

func (m TabsModel) Update(msg gothic.Msg, props TabsProps) (TabsModel, gothic.Cmd) {
	switch msg := msg.(type) {
	case TabClicked:
		m.Active = msg.Index
	}
	return m, nil
}
```

```templ
view Tabs(model TabsModel, props TabsProps) {
	<div class="tabs">
		for i, item := range props.Items {
			<button key={ i } class:active={ model.Active == i } onclick={ TabClicked{Index: i} }>
				{ item.Label }
			</button>
		}
	</div>
}
```

```templ
<Tabs items={ model.Tabs } />
```

A stateful component with no props uses `view Box(model BoxModel)`, `Init()`,
and `Update(msg)`.

Each instance owns its own state. The parent passes props but cannot read or
write child state. State starts at the zero value unless seeded from props.

Stateful components can render in static positions, inside `if` branches, and
as keyed list items. Closing a branch or removing a keyed list item unmounts the
component, so its state resets when it is created again. Stateful components
cannot accept children; use a pure component for layout.

#### Props and Init

`Update` may take the props struct. `Init` takes no parameters and runs after
the initial `PropsChanged`:

```go
func (m TabsModel) Init() gothic.Cmd
func (m TabsModel) Update(msg gothic.Msg, props TabsProps) (TabsModel, gothic.Cmd)
```

Props changes dispatch `gothic.PropsChanged`:

```go
case gothic.PropsChanged:
	if m.Active >= len(props.Items) {
		m.Active = len(props.Items) - 1
	}
```

Comparison is by reference. Assign new slices or structs instead of mutating in
place.

#### Seeding state from props

On mount, `PropsChanged` arrives with `Initial: true`. Later changes use
`Initial: false`. Seed local state only on the first message:

```go
case gothic.PropsChanged:
	if msg.Initial && props.Active != nil {
		m.Active = *props.Active
	}
```

The server applies `PropsChanged{Initial: true}` before rendering, then
serializes the resulting state. Client-created instances apply the same message
before their first render. Hydration reads the server state first. There is no
separate constructor.

#### Optional pointer props

```go
type TabsProps struct {
	Items  []Tab
	Active *int
}
```

```templ
<Tabs items={ model.Tabs } active={ model.ActiveTab } />
```

If the tag is omitted, the pointer is `nil` in Go and `null` in the client props
object. Use `*props.Active` after a nil check:

```templ
if props.Active != nil {
	<p>{ *props.Active }</p>
}
```

Address-of expressions such as `&model.Field` are not supported in views.

#### Component commands

`Init` and `Update` can return commands. Generated commands run in the
component's own loop and dispatch back to the component:

```go
func (m BoxModel) Update(msg gothic.Msg) (BoxModel, gothic.Cmd) {
	switch msg.(type) {
	case Load:
		return m, func() gothic.Msg {
			return Loaded{}
		}
	case Loaded:
		m.Status = "loaded"
	}
	return m, nil
}
```

Unmounting destroys the component. Late command results and prop updates are
ignored. Procedures remain App-owned; components cannot call them.

#### Actions

A component emits an action to the App with `emit`:

```go
type ButtonProps struct {
	Label   string
	Onclick gothic.Msg
}
```

```templ
view Button(model ButtonModel, props ButtonProps) {
	<button onclick={ emit(props.Onclick) }>{ props.Label }</button>
}
```

```templ
<Button label="Add todo" onclick={ AddClicked{} } />
```

`emit` is reserved and only available in component event expressions. The App
owns the message and any procedure it starts.

#### Nested stateful components

```templ
view Dashboard(model DashboardModel, props DashboardProps) {
	<section>
		<Counter label="static" onpick={ PickedChild{Label: "static"} } />
		if model.ShowBranch {
			<Counter label="branch" onpick={ PickedChild{Label: "branch"} } />
		}
		for i, item := range model.Items {
			<Counter key={ i } label={ item } onpick={ PickedChild{Label: item} } />
		}
	</section>
}
```

- Each child owns its own model.
- Props flow from the parent on every render.
- A child `emit` goes to the App.
- Branches and keyed list removal destroy children.
- Unmounting the parent destroys the whole tree.

## Markup

Elements, constant attributes, text, and interpolation work as in templ:

```templ
<div class="app">
	<p>Hello</p>
	<span>{ model.Count }</span>
</div>
```

Interpolation takes a Go expression parsed with `go/parser`.

### Attributes

```templ
<input value={ model.Draft } checked={ todo.Completed } />
```

Expression attributes update a DOM property or attribute. `class:name={ expr }`
adds or removes a class:

```templ
<li class:completed={ todo.Completed } class:editing={ editing } class:hidden={ hidden }>
```

`focus={ expr }` focuses the element when the expression becomes true.
`scroll={ expr }` scrolls the element into view when the expression becomes true.

Empty `aria-*` and `data-*` values are omitted from both the server render and
the client update. Other attributes keep their values, so `value=""` still
renders.

`key={ expr }` declares a list key. `key` is reserved and is not passed to
components as a prop.

## Lists

```templ
for _, item := range model.Items {
	<li key={ item.ID }>{ item.Text }</li>
}
```

The range source is a Go expression, normally a model field. Add `key={ expr }`
to the root of the loop body, or to a component call, to enable keyed list
reconciliation. Keyed lists reuse nodes by key and update, reorder, or remove
them as the model changes.

Lists without a key append and remove from the end only.

The server renders the initial children. The client adopts them and hydrates
their events.

## Conditionals

```templ
if len(model.Todos) > 0 {
	<ul>...</ul>
} else if model.Loading {
	<p>Loading todos.</p>
} else {
	<p class="empty">Nothing to do yet.</p>
}
```

The condition is a Go expression in the client Go subset. `else if` is sugar
for a nested conditional.

- The server renders only the chosen branch.
- The client adopts it and swaps branches when the condition changes.
- `len(x)` is supported and counts slices, arrays, strings, and map keys.
- A branch can contain stateful components. They mount when the branch appears
  and unmount when it leaves.
- A pure component still renders exactly one root element, so an `if` cannot be
  the component root.
- A branch-level `for` must have one element as its body.
- `switch` is not part of the view language.

Text that begins with the word `if` is parsed as a conditional. Rephrase such
prose or place it inside an element.

## Events

An event attribute is a Go expression that produces a message:

```templ
<button onclick={ PingClicked{} }>Ping</button>
<button onclick={ DeleteClicked{ID: item.ID} }>Delete</button>
```

The generated JavaScript dispatches the message value. There is no imperative
handler block.

Event handlers receive the DOM event as `event`:

```templ
<input oninput={ DraftChanged{Text: event.target.value} } />
```

`onenter` and `onescape` are shortcuts for keydown handlers:

```templ
<input onenter={ SaveEdit{ID: todo.ID} } onescape={ CancelEdit{ID: todo.ID} } />
```

`onkeynav` is a keydown handler for ArrowUp, ArrowDown, Home, End, PageUp,
PageDown, Enter, and Escape. It calls `preventDefault` before dispatching:

```templ
<div onkeynav={ KeyNav{Key: event.key} }>...</div>
```

## Routing

Define a `LocationChanged` message:

```go
type LocationChanged struct {
	Path  string
	Query string
	Hash  string
}
```

The client dispatches it on load and on `popstate`. Mark internal anchors with
`route`:

```templ
<a href="/about" route>About</a>
```

`route` is only supported on `<a>` elements and only in the App view.

Path patterns and parameters come from the `route` package:

```go
params := route.Params("/todos/:id", msg.Path)
if params != nil {
	m.ID = params["id"]
}
```

`route.Params` returns `map[string]string` or `nil`. Patterns are
slash-separated, `:` captures one non-empty segment, and the pattern and path
must have the same number of segments.

Programmatic navigation comes from the `browser` module:

```go
browser.Navigate("/about")
browser.Replace("/about")
browser.Back()
```

These are client-only and valid in `Init`, `Update`, or commands. They require a
`LocationChanged` message.

`MountApp` registers a catch-all below the pattern, so deep links reach the app.
Use a request-scoped model to render the right state:

```go
s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
	return NewModel(r.URL.Path)
}))
```

See [Routing and documents](guide/04-routing.md) for a full walkthrough.

## Documents

`MountApp` renders a full HTML document. The zero `Document` gets a title of
`Gothic App`, no icons, and no image. Set server-wide defaults with
`SetDocument`, then override them for a mount with `WithDocument`:

```go
s := gothic.NewServer()
s.SetDocument(gothic.Document{SiteName: "Gothic"})
s.MountApp("/", gothic.NewApp(NewModel()),
	gothic.WithDocument(gothic.Document{
		Title:       "Gothicbook",
		Description: "A tiny design system demo.",
		Image:       "/og.png",
		Icons:       []gothic.Icon{{Href: "/favicon.svg"}},
		Styles:      []string{"/style.css"},
	}),
)
```

- `WithDocument` replaces the server document for that mount. Zero fields fall
  back to the built-in defaults.
- Leave `Image` empty to omit `og:image`. Leave `Icons` nil to omit icons.
- Relative `og:image` and `og:url` values become absolute using the request host
  and forwarding headers.
- `gothic.WriteDocument(w, r, doc, body)` renders the same shell around any
  component, without a Model.
- `/_gothic/` is reserved for the runtime and generated modules.

## Browser interop

The `browser` module is a curated, client-only surface:

```go
import "github.com/metruzanca/gothic/browser"

browser.Alert("Saved")
browser.Log("loaded", count)
browser.SetTitle("Todos")
browser.SetItem("theme", "dark")
theme := browser.GetItem("theme")
browser.RemoveItem("theme")
browser.Copy("text to copy")
browser.Focus("#search")
height := browser.ClientHeight("#rows")
browser.Reload()
```

Calls are lowered to browser APIs in the generated JavaScript. They are only
valid in `Init`, `Update`, or commands; calling them from a view is a compile
error. The package does not expose arbitrary `window` access.

## Client Go subset

`Init`, `Update`, commands, and component views compile a growing subset of Go:

```go
item := m.Items[0]
m.Items[0] = item
page := m.Items[1:3]
tail := m.Items[1:]
m.Counts = map[string]int{"open": 1, "done": 2}
m.Total = len(m.Counts)
delete(m.Counts, "done")
m.Total += 1
m.Count--
```

Supported:

- field access, indexing, and two-index slicing,
- map literals, `len`, `delete`, and map indexing,
- `append`, assignments, and compound assignments,
- `++` and `--`,
- `if` and `if err != nil`,
- range loops over slices and maps with `break` and `continue`,
- non-type switches, including tagless switches and init statements,
- procedure calls and command closures.

Not supported:

- three-index slices,
- `&^=`,
- labeled branches,
- `goto`,
- `fallthrough`,
- declaring variables in view expressions.

Differences from server Go:

- `dom.Px(n)` formats pixel lengths for style attributes.
- `dom.Str(*string)`, `dom.StrOr(*string, string)`, and `dom.True(*bool)` read
  optional props.
- `dom.If(cond, yes, no)` and `dom.ID(base, index)` build attribute values and
  ARIA ids.
- `dom.Ptr(value)` and `dom.BoolPtr(value)` create optional pointer props.
- `strings.ToLower`, `strings.TrimSpace`, and `strings.Contains` work in
  `Update` and commands.
- `browser.ClientHeight(selector)` reads an element height.
- A missing map key reads as `undefined`, not the zero value.
- Map iteration uses JavaScript insertion order, and map keys arrive as strings.
- `/=` is floating division.
- Slices copy instead of sharing a backing array.

## Procedures

Declare client-callable procedures as an interface that embeds
`gothic.RPCService`:

```go
type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
	AddTodo(ctx context.Context, text string) error
	ListTodos(ctx context.Context) ([]Todo, error)
}
```

The first parameter must be `context.Context`. Remaining parameters are sent by
the client as JSON. The result is `(T, error)` or `error`.

The generator emits:

- a registrar `RegisterProcedures(s, impl)`,
- a client-side Go stub used to compile commands,
- a JavaScript RPC function.

Commands call procedures directly:

```go
func callPing() gothic.Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}
```

The Go stub panics if called on the server. The server never runs commands, so
this is safe. An interface without `gothic.RPCService` is an ordinary Go
interface.

## Commands

A command performs work and returns a message:

```go
func loadItems() gothic.Msg {
	items, err := ListItems()
	if err != nil {
		return LoadFailed{Message: err.Error()}
	}
	return Loaded{Items: items}
}
```

`Update` returns it as the second value. A command can also be a closure:

```go
return m, func() gothic.Msg {
	if err := AddTodo(m.Draft); err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return loadTodos()
}
```

Compose commands with `gothic.Batch` or `gothic.Sequence`:

```go
return m, gothic.Batch(loadItems(), loadStats())

return m, gothic.Sequence(
	func() gothic.Msg { return FirstDone{} },
	func() gothic.Msg { return SecondDone{} },
)
```

- `Batch` starts commands concurrently.
- `Sequence` runs them in order.
- Each non-nil result is dispatched.
- Nil commands are ignored.
- Component commands stop after unmount.

Both are client-only. The Go stubs exist so `Init`, `Update`, and commands
compile.

A stateful component can send a message to the App from `Init`, `Update`, or a
command with `gothic.Emit`:

```go
return m, gothic.Emit(ValueChanged{Value: m.Value})
```

`gothic.Emit` is a component-only command. Using it in the App is a compile
error.

## Rendering

- The server renders the initial model and serializes it into the page.
- Components are expanded into the app before either target is generated.
- Interpolated text is wrapped in comment markers. The client updates the text
  node in place.
- The client reads the model, adopts the server-rendered list nodes, hydrates
  their events, and binds event attributes.
- On dispatch, the runtime calls `Update`, replaces the model, updates text and
  lists, and runs the returned command.
- Rendering is direct DOM updates. There is no virtual DOM.

## Current limits

Supported:

- one application model per package,
- one `App` view with one model parameter,
- pure components with a props struct,
- stateful components with their own model, props struct, `Init`, `Update`, and
  view,
- named tag props mapped case-insensitively,
- optional pointer props with `nil` and `null` semantics,
- children through `gothic.Children` on pure components,
- components in the same file or separate files in the same package,
- nested pure and stateful components,
- stateful components inside conditionals and keyed lists, with unmount,
- `gothic.PropsChanged` with `Initial` for prop-seeded state,
- component commands with unmount guards,
- `if`, `else if`, and `else`,
- `for` lists with keyed reconciliation,
- message-valued events and `event` values,
- dynamic attributes, class toggles, and focus,
- indexing, slicing, maps, `len`, `delete`, compound assignment, `++`/`--`,
- range loops and non-type switches,
- `append` and assignments,
- `if err != nil`,
- commands with procedure calls and closures,
- `Batch` and `Sequence`,
- procedure arguments,
- path, query, and hash state through `LocationChanged`,
- `route` links and `route.Params`,
- request-scoped models and catch-all mounting,
- documents, metadata, icons, stylesheets, and previews,
- curated browser interop through `browser`,
- headless long lists through `virtual`.

Not supported yet:

- component-declared procedures; procedures are App-only,
- address-of expressions in views,
- controlled components and function-valued props,
- multiple app models,
- Go-callable component functions,
- `switch` in views,
- multiple `App` parameters,
- typed procedure argument validation,
- arbitrary browser globals outside `browser`,
- wildcard or optional route segments and reverse URL building,
- message serialization beyond JSON,
- `go/packages` type information,
- formatter, LSP, and source maps.
