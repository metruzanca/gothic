# 03 Components

A gothic view can be one big function, but most apps split it into components.
There are two kinds:

- **Pure components** are stateless functions of their props.
- **Stateful components** own a small model with their own `Init` and `Update`.

Both live in `.gth` files. Pure components can live in the same file as the App
or in a separate `.gth` file in the same package. A separate component file
contains only a package clause and views, so declare its props struct in the
app prelude. Stateful components live in the app file.

## Pure components

A pure component takes one parameter named `props`, backed by a props struct:

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

- Tag names map to struct fields case-insensitively.
- `text` maps to `Text`.
- A tag without a value passes `true`.
- Non-pointer fields are required.
- Pointer fields are optional. Omitting the tag leaves the field `nil`.

```templ
type BadgeProps struct {
	Text  string
	Tone  *string
}
```

```templ
<Badge text="ready" />
<Badge text="ready" tone="warning" />
```

Pure components render exactly one root element and should reference only their
props and package-level declarations. They cannot declare state and cannot call
procedures.

## Children

A component accepts children when its props struct embeds `gothic.Children`:

```templ
import "github.com/metruzanca/gothic"

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

Use it like any other element:

```templ
<Field label="Your name">
	<input value={ model.Name } oninput={ NameChanged{Text: event.target.value} } />
</Field>
```

- The component must render `{ props.Children }` exactly once.
- Children are expanded in the caller's scope, so they can use the App model,
  loop variables, events, conditionals, and stateful components.
- Children render nothing when the component is called without children.
- Named slots and fallback content are not supported.

A named field works too:

```templ
type FieldProps struct {
	Label    string
	Children gothic.Children
}
```

## Stateful components

A stateful component is a view whose first parameter is a model with `Init` and
`Update`. A second parameter is an optional props struct:

```go
type Tab struct {
	Label string
	Panel string
}

type TabsProps struct {
	Items []Tab
}

type TabsModel struct {
	Active int
}

type TabClicked struct{ Index int }

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

Call it with named props:

```templ
<Tabs items={ model.Tabs } />
```

Each instance owns its own state. The parent passes props but cannot read or
write child state.

Stateful components can render in static positions, inside `if` branches, and
as keyed list items. A component inside a branch is unmounted when the branch
leaves the DOM. Re-entering the branch creates a fresh instance, so its state
resets. Removing a keyed list item does the same.

Components with no props use `view Box(model BoxModel)`, `Init()`, and
`Update(msg)`.

## Props and lifecycle

Props reach the component through its view and its `Update` method. When a prop
reference changes, the runtime dispatches `gothic.PropsChanged`:

```go
func (m TabsModel) Update(msg gothic.Msg, props TabsProps) (TabsModel, gothic.Cmd) {
	switch msg := msg.(type) {
	case TabClicked:
		m.Active = msg.Index
	case gothic.PropsChanged:
		if m.Active >= len(props.Items) {
			m.Active = len(props.Items) - 1
		}
	}
	return m, nil
}
```

Comparison is by reference. Assign new slices or structs instead of mutating
them in place.

### Seeding state from props

On mount, `PropsChanged` arrives with `Initial: true`. Later changes use
`Initial: false`. Seed local state only when `msg.Initial`:

```go
type TabsModel struct {
	Active int
}

func (m TabsModel) Update(msg gothic.Msg, props TabsProps) (TabsModel, gothic.Cmd) {
	switch msg := msg.(type) {
	case gothic.PropsChanged:
		if msg.Initial && props.Active != nil {
			m.Active = *props.Active
		}
	}
	return m, nil
}
```

- The server applies `PropsChanged{Initial: true}` before rendering and
  serializes the resulting state.
- Client-created instances apply the same message before their first render.
- Hydration reads the server-serialized state first.
- There is no separate constructor.

`Init` may also take the props struct when it needs to start a command from
prop values:

```go
func (m TabsModel) Init(props TabsProps) gothic.Cmd {
	return nil
}
```

### Optional pointer props

A pointer field is optional. Omit the tag and the component sees `nil`:

```go
type TabsProps struct {
	Items  []Tab
	Active *int
}
```

```templ
<Tabs items={ model.Tabs } active={ model.ActiveTab } />
```

```templ
if props.Active != nil {
	<p>{ *props.Active }</p>
}
```

Address-of expressions such as `&model.Field` are not supported in views. Store
a pointer-valued field in the model or pass `nil`.

## Component commands

`Init` and `Update` can return commands. Generated commands run in the
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

Unmounting destroys the component. Command results that resolve after unmount
are ignored, and prop updates are ignored.

Procedures remain App-owned. A component cannot call an App procedure. Emit an
action to the App and let the App run the procedure.

## Actions

A stateful component can send a message to the App with `emit`:

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

The App passes the message and handles it:

```templ
<Button label="Add todo" onclick={ AddClicked{} } />
```

`emit` is only available in component event expressions. It sends the message
straight to the App, where `Update` runs procedures and commands. There is no
separate child-to-parent channel.

## Nesting and lists

A stateful component can render another stateful component in a static
position, an `if` branch, or a keyed list:

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
- Closing a branch destroys its children, so their state resets.
- Removing a keyed list item destroys its child.
- Unmounting the parent destroys the whole tree.
- Keyed list children keep their state while their key stays in the list.

A stateful component can also take children through `gothic.Children`:

```go
type PanelProps struct {
	gothic.Children
	Title string
}
```

```templ
view Panel(model PanelModel, props PanelProps) {
	<section>
		<h2>{ props.Title }</h2>
		{ props.Children }
	</section>
}
```

Children keep the App scope, so App state and events work inside the panel.

## Conditionals and loops

Components can use `if`, `else if`, `else`, and `for` like the App view:

```templ
view Badge(props BadgeProps) {
	<span>
		if props.Active {
			<b class="on">{ props.Text }</b>
		} else {
			<i class="off">{ props.Text }</i>
		}
	</span>
}
```

A pure component still renders exactly one root element, so an `if` cannot be
the component root. A branch-level `for` must have one element as its body.

## The UI kit

The optional UI kit is a separate Go module with pure and stateful components:

```sh
go get github.com/metruzanca/gothic/ui
```

Add the component package to the app prelude:

```go
//gothic:components github.com/metruzanca/gothic/ui
```

Mount the styles in `main.go`:

```go
import "github.com/metruzanca/gothic/ui"

ui.MountStyles(s)
```

Use namespaced tags:

```templ
<ui.Button label="Save" primary={ true } onclick={ SaveClicked{} } />
<ui.Card title="Card title" body="Card body" />
<ui.Field label="Your name">
	<input value={ model.Name } oninput={ NameChanged{Text: event.target.value} } />
</ui.Field>
<ui.Tabs items={ model.Tabs } />
<ui.Disclosure item={ model.FeatureDisclosure } />
```

Pure UI kit components are expanded into your app by the generator. Stateful
ones ship as a precompiled module. `ui.Button` emits its `onclick` message to
the App, which owns the procedure.

See [ui/README.md](../../ui/README.md) for the current component list.

## Limits

- Components cannot declare procedures; procedures are App-owned.
- Address-of expressions are not supported in views.
- Named slots and fallback content are not supported.
- Function-valued props are not supported.
- Go cannot call a component function directly yet.
- `switch` is not part of the view language.
- A pure component renders exactly one root element.
- Precompiled library stateful components cannot accept children.

## What next

Components cover the view layer. The next chapter adds URL state and the HTML
document around your app:

- [Routing and documents](04-routing.md)
- [Syntax reference: components](../SYNTAX.md#views)
