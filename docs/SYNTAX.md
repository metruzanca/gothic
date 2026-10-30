# gothic syntax

gothic is currently a proof of concept. The application model follows
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and The Elm
Architecture. A `.gth` file contains ordinary Go followed by one or more
views. One view, `App`, is the root. The others are pure components.
`main.go` remains the entrypoint.

## Model-View-Update

The application is a Go type that implements:

```go
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Component
}
```

Following Bubble Tea:

- `Msg` is `any`. Messages are ordinary Go values and are handled with a type
  switch.
- `Cmd` is `func() Msg`. A command performs work and returns a message.
- `Update` is a method on the model and returns the next model and an optional
  command.
- `Init` returns an optional command.
- `View` is generated from the `.gth` view.

## Example

`pingpong.gth` holds the Go prelude, the view, and the procedure contract:

```gothic
package pingpong

import (
	"context"

	"github.com/metruzanca/gothic"
)

type Model struct {
	Messages []string
}

func NewModel() Model {
	return Model{Messages: []string{"ready"}}
}

type PingClicked struct{}
type PongReceived struct{ Text string }
type PingFailed struct{ Message string }

type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}

func (m Model) Init() gothic.Cmd {
	return nil
}

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case PingClicked:
		m.Messages = append(m.Messages, "ping")
		return m, callPing
	case PongReceived:
		m.Messages = append(m.Messages, msg.Text)
	case PingFailed:
		m.Messages = append(m.Messages, msg.Message)
	}
	return m, nil
}

func callPing() gothic.Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}

view App(model Model) {
	<div class="app">
		<button onclick={ PingClicked{} }>Ping</button>
		<ul>
			for _, m := range model.Messages {
				<li>{ m }</li>
			}
		</ul>
	</div>
}
```

`main.go` stays the entrypoint:

```go
type procedures struct{}

func (procedures) Ping(ctx context.Context) (string, error) {
	return "pong", nil
}

func main() {
	s := gothic.NewServer()
	RegisterProcedures(s, procedures{})
	s.MountApp("/", gothic.NewApp(NewModel()))
	s.ListenAndServe(":8080")
}
```

The generator emits:

- `pingpong_gothic.go`: the Go prelude, the view, the generated `View` method,
  the registrar, procedure client stubs, and the embedded module.
- `pingpong_gothic.js`: the compiled app and view module.

## File layout

A `.gth` app file has two sections:

1. A Go prelude: package, imports, model, messages, `Init`, `Update`,
   commands, and procedure interfaces.
2. One or more views, starting with a line that begins with `view`.

Everything before the first top-level `view` line is parsed as Go. Everything
from that line onward is parsed as gothic markup.

The app view is named `App`:

```gothic
view App(model Model) {
	...
}
```

Any other view is a component:

```gothic
view Badge(text string) {
	<span class="badge">{ text }</span>
}
```

Components can live in the same app file or in a separate `.gth` file in the
same directory. A separate component file contains only a package clause and
views. It has no model and no `Init` or `Update`.

## View language

### View signature

```gothic
view App(model Model) {
	...
}
```

The app view is a pure function of the model. The model parameter name and type
are used to generate the Go `View` method and the JavaScript renderer.

### Pure components

Pure components can declare a props struct:

```go
type BadgeProps struct {
	Text   string
	Hidden bool
}

view Badge(props BadgeProps) {
	<span class="badge" class:hidden={ props.Hidden }>{ props.Text }</span>
}
```

Invoke a component with a capitalized tag. Named tags map to exported struct
fields case-insensitively:

```gothic
<Badge text={ model.Status } />
<Badge text="ready" hidden />
```

- `text` maps to `Text`.
- `hidden` maps to `Hidden` and passes `true`.
- Non-pointer fields are required.
- Pointer fields are optional. Omitting the tag leaves the field nil.

Older pure components can still declare individual props:

```gothic
view Badge(text string) {
	<span class="badge">{ text }</span>
}
```

New components should prefer a props struct. Pure components with a props
struct use one parameter named `props`.

The compiler expands components into the app before generating Go and
JavaScript. Props are substituted as a struct literal.

Rules for pure components:

- They are stateless. They cannot declare `state`.
- They render exactly one root element.
- They should reference only their props and package-level declarations.
- They may declare one default `<slot />` to receive children.
- Components cannot be called from Go code yet.
- Capitalized tags are always components, so camelCase SVG tags are not
  supported yet.

#### Component children

A pure component can receive children and place them with one `<slot />`:

```go
type FieldProps struct {
	Label string
}

view Field(props FieldProps) {
	<div class="ui-field">
		<span class="ui-field-label">{ props.Label }</span>
		<slot />
	</div>
}
```

```gothic
<ui.Field label="Your name">
	<input value={ model.Name } oninput={ NameChanged{Text: event.target.value} } />
</ui.Field>
```

Children are expanded in the caller's scope, so they can use the App model,
loop variables, events, conditionals, loops, and stateful components. The
component supplies layout only. A slot renders nothing when the component is
called without children.

Rules for children and slots:

- one default slot per pure component,
- `<slot />` cannot take attributes or children,
- a slot cannot be the component root,
- named slots and fallback content are not supported yet,
- stateful components cannot accept children yet,
- children on a component without a slot are an error,
- a slot outside a pure component is an error,
- `<slot />` is reserved and cannot be used as an HTML element.

### Stateful components

A stateful component is a view whose first parameter is a model with `Init` and
`Update` methods. A second parameter is a props struct:

```go
type TabsProps struct {
	Items []Tab
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
	<div class="tabs">
		for i, item := range props.Items {
			<button key={ i } class:active={ model.Active == i } onclick={ TabClicked{Index: i} }>
				{ item.Label }
			</button>
		}
	</div>
}
```

Call sites still use named tags:

```gothic
<Tabs items={ model.Tabs } />
```

Stateful components require a props struct when they take props. Components
with no props keep `view Box(model BoxModel)`, `Init()`, and `Update(msg)`.

Each instance owns its own state. The parent passes props but does not read or
write child state. State starts at the model zero value. Props are passed to the
view, `Init`, and `Update`.

Stateful components can render in static positions, inside `if` branches, and
as keyed list items. A list-created component may contain its own loops and
conditionals, and must be the loop body root when used in a list.

A component inside a conditional branch is unmounted when the branch leaves the
DOM. Re-entering the branch creates a fresh instance, so component state resets.
The same reset applies when a keyed list removes an item and later creates it
again.

```gothic
if model.ShowDetails {
	<ui.Disclosure item={ model.FeatureDisclosure } />
}

for _, item := range model.Items {
	<ui.Disclosure key={ item.ID } item={ item } />
}
```

Stateful components in this phase:

- support static positions, conditional branches, and keyed lists,
- must be the loop body root when used in a list,
- may contain their own loops and conditionals,
- take a props struct in the view, `Init`, and `Update`,
- run `Init` once on mount,
- can run commands from `Init` and `Update`,
- cannot declare procedures or call App procedures,
- can emit messages to the parent from event attributes,
- cannot contain other stateful components,
- reset their state when unmounted by a branch swap or keyed list removal.

#### Props structs

A props struct uses exported Go fields. Tag names match fields
case-insensitively.

```go
type TabsProps struct {
	Items  []Tab
	Active *int
}
```

- Non-pointer fields are required.
- Pointer fields are optional. If the tag is omitted, the field is nil in Go
  and `null` in the client props object.
- A pointer value is passed as its value or `null`, not as a pointer object.

```gothic
<Tabs items={ model.Tabs } active={ model.ActiveTab } />
```

If `model.ActiveTab` is nil, the component sees `props.Active == nil` and can
use a default.

Use `*props.Active` after a nil check:

```gothic
if props.Active != nil {
	<p>{ *props.Active }</p>
}
```

Address-of expressions such as `&model.Field` are not supported in views. Store
a pointer-valued field in the model or pass `nil`.

#### Props in Init and Update

`Init` and `Update` can take the props struct:

```go
func (m TabsModel) Init(props TabsProps) gothic.Cmd
func (m TabsModel) Update(msg gothic.Msg, props TabsProps) (TabsModel, gothic.Cmd)
```

Components with no props use `Init()` and `Update(msg)`.

When any prop reference changes, the component dispatches
`gothic.PropsChanged`. Handle it in `Update` to react:

```go
switch msg.(type) {
case gothic.PropsChanged:
	if m.Active >= len(props.Items) {
		m.Active = len(props.Items) - 1
	}
}
```

Comparison is by reference, so assign new slices or structs instead of
mutating in place.

`Init` cannot seed the initial model with props yet. Component state still
starts at the zero value. Use props directly in the view or `Update` until a
prop-aware model constructor exists.

#### Component commands

Component `Init` and `Update` can return commands. Generated commands run in the
component's own loop and dispatch their result back to the component:

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

Procedures remain App-owned. Component code cannot call an App procedure. Emit
an action to the App instead, and let the App run the procedure.

Unmounting destroys the component. Command results that resolve after unmount
are ignored, and prop updates are ignored.

Emit a parent message with `emit` in a component event:

```go
type ButtonProps struct {
	Label   string
	Onclick gothic.Msg
}
```

```gothic
view Button(model ButtonModel, props ButtonProps) {
	<button onclick={ emit(props.Onclick) }>{ props.Label }</button>
}
```

The parent passes the message:

```gothic
<ui.Button label="Add todo" onclick={ AddClicked{} } />
```

The App owns `AddClicked` in its `Update`, and runs any commands or server
procedures there. Components do not own server procedures. `emit` is reserved
and only available to component event expressions.

The installable UI kit is precompiled and uses this model. Add component
packages with a directive in the app prelude:

```go
//gothic:components github.com/metruzanca/gothic/ui
```

Use namespaced tags for external components:

```gothic
<ui.Button label="Save" primary={ true } onclick={ SaveClicked{} } />
<ui.Tabs items={ model.Tabs } />
```

The compiler reads the package manifest and mounts each external component as a
self-contained root.

### Markup

Elements, constant attributes, text, and interpolation work as in templ:

```gothic
<div class="app">
	<p>Hello</p>
	<span>{ model.Count }</span>
</div>
```

Interpolation takes a Go expression. It is parsed with `go/parser`.

### Lists

```gothic
for _, item := range model.Items {
	<Item key={ item.ID } item={ item } />
}
```

The range source is a Go expression, normally a model field. Add `key={ expr }`
to the root of the loop body, or to a component call, to enable keyed list
reconciliation. Keyed lists reuse nodes by key, update their text, and reorder
or remove them as the model changes. Lists without a key append and remove from
the end only. `key` is reserved and is not passed to components as a prop.

The server renders the initial children. The client adopts them and hydrates
their events.

### Conditionals

```gothic
if len(model.Todos) > 0 {
	<ul>
		for _, todo := range model.Todos {
			<li>{ todo.Title }</li>
		}
	</ul>
} else if model.Loading {
	<p>Loading todos.</p>
} else {
	<p class="empty">Nothing to do yet.</p>
}
```

The condition is a Go expression in the same subset as `Update` and events.
`else if` is sugar for a nested conditional. A branch can hold any number of
nodes, including text, interpolation, elements, loops inside an element,
components, and nested conditionals.

The server renders only the chosen branch. The client adopts it on load and
swaps branches when the condition changes, removing the unused nodes from the
DOM. `len(x)` is supported in conditions and compiles to a null-safe length
read, so `len(model.Items) > 0` works when the serialized slice is `null`.

A branch can contain stateful components. They mount when the branch appears
and unmount when it leaves, so their state resets on re-entry. A `for` loop can
be the direct child of a branch when the loop body is one element.

Limits in this iteration:

- a pure component still renders exactly one root element, so an `if` cannot be
  the component root,
- a branch-level loop body must be one element,
- `switch` is not part of the view language.

As with `for`, text that begins with the word `if` is parsed as a conditional.
Rephrase such prose or place it inside an element that makes the intent
unambiguous.

### Events

An event attribute is a Go expression that produces a message:

```gothic
<button onclick={ PingClicked{} }>Ping</button>
<button onclick={ DeleteClicked{ID: item.ID} }>Delete</button>
```

The generated JavaScript dispatches the message value. There is no imperative
handler block in the application model.

Event handlers receive the DOM event as `event`. This is useful for inputs:

```gothic
<input oninput={ DraftChanged{Text: event.target.value} } />
```

`onenter` and `onescape` are shortcuts for keydown handlers:

```gothic
<input onenter={ SaveEdit{ID: todo.ID} } onescape={ CancelEdit{ID: todo.ID} } />
```

### Attributes and classes

Expression attributes update a DOM property or attribute:

```gothic
<input value={ model.Draft } checked={ todo.Completed } />
```

`class:name={ expr }` adds or removes a class when the expression changes:

```gothic
<li class:completed={ todo.Completed } class:editing={ editing } class:hidden={ hidden }>
```

`focus={ expr }` focuses the element when the expression becomes true.

### Routing

For path-based URL state, define a `LocationChanged` message:

```go
type LocationChanged struct {
	Path  string
	Query string
	Hash  string
}
```

The client dispatches it on load and on `popstate`. `Path` is
`window.location.pathname`, `Query` is `window.location.search` including the
leading `?`, and `Hash` is `window.location.hash` including `#`.

Mark internal anchors with `route`:

```gothic
<a href="/about" route>About</a>
```

A plain left click is intercepted, `history.pushState` runs, and
`LocationChanged` dispatches. Modifier clicks, `target` values other than
`_self`, `download` links, and external origins are left to the browser.

Programmatic navigation comes from the `js` package:

```go
js.Navigate("/about")
js.Replace("/about")
js.Back()
```

These are client-only and valid in `Init`, `Update`, or commands. They require
a `LocationChanged` message.

Render the right state for deep links with a request-scoped model:

```go
s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
	return NewModel(r.URL.Path)
}))
```

`MountApp` also registers a catch-all below the pattern, so paths below the
mount reach the app. The server renders the model for the request, then the
client hydrates it and dispatches `LocationChanged` on load.

`HashChanged` still works for hash-only apps. If an app defines both
`LocationChanged` and `HashChanged`, `LocationChanged` takes precedence.

### Browser interop

The `js` package is a curated, client-only surface:

```go
import "github.com/metruzanca/gothic/js"

js.Alert("Saved")
js.Log("loaded", count)
js.SetTitle("Todos")
js.SetItem("theme", "dark")
theme := js.GetItem("theme")
js.RemoveItem("theme")
js.Copy("text to copy")
js.Focus("#search")
js.Reload()
```

Calls are lowered to browser APIs in the generated JavaScript. They are only
valid in `Init`, `Update`, or commands; calling them from a view is a compile
error. The package does not expose arbitrary `window` access.

### Procedures

Declare client-callable procedures as a Go interface that embeds
`gothic.RPCService`:

```go
type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
	AddTodo(ctx context.Context, text string) error
	ToggleTodo(ctx context.Context, id int64) error
	ListTodos(ctx context.Context) ([]Todo, error)
}
```

The interface name is arbitrary. The compiler looks for the embedded
`gothic.RPCService`, then generates the registrar and the client stubs from the
interface methods.

The generator emits:

- a registrar `RegisterProcedures(s, impl)`,
- a client-side Go stub used to compile commands,
- a JavaScript RPC function.

The first parameter must be `context.Context`. Remaining parameters are
arguments sent by the client as JSON. The result must be `(T, error)` or
`error`. The client stub drops the context.

Commands call procedures directly:

```go
func callPing() gothic.Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}

func callAdd(text string) gothic.Msg {
	if err := AddTodo(text); err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return TodosLoaded{}
}
```

The Go stub panics if it is called on the server. The server never runs
commands, so this is safe for the proof of concept.

An interface without `gothic.RPCService` is an ordinary Go interface and is not
treated as a procedure contract.

## Commands

A command is a function that returns a message:

```go
func loadItems() gothic.Msg {
	items, err := ListItems()
	if err != nil {
		return LoadFailed{Message: err.Error()}
	}
	return Loaded{Items: items}
}
```

`Update` returns it as the second value:

```go
return m, loadItems
```

A command can also be a closure, which is useful when it needs values from the
message or model:

```go
return m, func() gothic.Msg {
	if err := AddTodo(m.Draft); err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return loadTodos()
}
```

The runtime renders the new model first, then runs the command. When the
command returns a message, the runtime dispatches it. This keeps optimistic
updates visible before the asynchronous work completes.

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

The proof of concept supports:

- one application model per package,
- one `App` view with a single model parameter,
- pure components with a props struct, or individual props during migration,
- stateful components with their own model, a props struct, Init, Update, and view,
- named tag props mapped case-insensitively to props struct fields,
- optional pointer props with nil and null semantics,
- props in component Init and Update,
- app-local stateful components and precompiled external components,
- components in the same file or separate files in the same package,
- nested pure components and components inside lists,
- children and a single default `<slot />` in pure components,
- stateful components inside slotted content,
- `if`, `else if`, and `else` in views, including inside loops and pure
  components,
- stateful components inside conditional branches, with unmount and state reset,
- stateful components as keyed list items, including components with internal
  loops,
- `gothic.PropsChanged` on prop changes,
- component Init running once on mount,
- component commands with unmount guards,
- `for` loops directly inside conditional branches when the body is one element,
- the installable gothic UI kit,
- message structs with zero or simple fields,
- `Update` with a type switch,
- `append` and assignments,
- `if err != nil`,
- commands with procedure calls and command closures,
- procedure arguments,
- lists over a model field and keyed list reconciliation,
- reactive interpolated text and message-valued event bindings,
- event values such as `event.target.value`,
- dynamic attributes, class toggles, and focus bindings,
- hash-based route changes through a `HashChanged` message,
- path, query, and hash URL state through a `LocationChanged` message,
- `route` links with pushState, replaceState, back, and popstate,
- request-scoped app models and catch-all app mounting for deep links,
- curated browser interop through the `js` package,
- component-to-parent action messages through `emit`,
- App-owned procedures passed to components as action props,
- procedure contracts through an interface embedding `gothic.RPCService`.

Not supported yet:

- component-declared procedures; procedures are App-only,
- prop-seeded component initial state; component models still start at the zero value,
- address-of expressions in views,
- controlled components and function-valued props,
- stateful components nested inside other stateful components,
- multiple app models,
- children on stateful components,
- named slots and fallback slot content,
- Go-callable component functions,
- `switch` in views,
- multiple `App` parameters,
- `Batch` and `Sequence`,
- typed procedure argument validation,
- arbitrary browser globals outside the `js` package,
- path parameters, nested routers, and route matching helpers,
- message serialization,
- `go/packages` type information.

The earlier `state`, `rpc`, and imperative handler syntax still exists in the
compiler as a legacy path, but it is not the direction of the proof of concept.
