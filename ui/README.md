# gothic UI kit

A small, proof-of-concept design system for gothic. It ships precompiled Go
renderers and JavaScript modules in a separately versioned Go module.

## Install

```sh
go get github.com/metruzanca/gothic/ui
```

## Use

Add the component package to the app prelude:

```go
//gothic:components github.com/metruzanca/gothic/ui
```

Import the package in `main.go` and mount the styles:

```go
import "github.com/metruzanca/gothic/ui"

s := gothic.NewServer()
ui.MountStyles(s)
```

Use namespaced tags in `.gth` views:

```gothic
<ui.Button label="Save" primary={ true } onclick={ SaveClicked{} } />
<ui.Button label="Cancel" primary={ false } onclick={ nil } />
<ui.Card title="Card title" body="Card body" />
<ui.Tabs items={ model.Tabs } />
<ui.Accordion items={ model.Accordion } />
```

`Button` emits its `onclick` message to the parent app through `emit`. The app
owns the message handler and any server procedures it calls.

`Tabs` uses `[]ui.Tab` and `Accordion` uses `[]ui.AccordionItem`.

## Components

- `Button` with `label`, `primary`, and `onclick gothic.Msg`
- `Card` with `title` and `body`
- `Tabs` with `items []Tab`
- `Accordion` with `items []AccordionItem`

## Development

The generated files are committed:

- `ui_gothic.go`
- `ui_gothic.js`
- `gothic-components.json`

Regenerate them from the repository root with:

```sh
go run ./cmd/gothic generate ./ui
```

The UI module needs the core release that ships stateful components and the
component runtime helpers. During local development, `ui/go.work` builds the UI
module against this checkout. The `require` entry in `ui/go.mod` should be
bumped when the next core release is tagged.

## CSS

`ui.MountStyles(s)` serves the kit stylesheet at `/gothic-ui.css`.
