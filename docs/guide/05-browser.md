# 05 Browser interop

Gothic apps do not need to write JavaScript, but sometimes an app needs a browser
API. The `browser` module is a small, curated Go surface for that. Every
function is client-only and has a direct lowering in the generated JavaScript.

The module is separate from core gothic:

```sh
go get github.com/metruzanca/gothic/browser
```

```go
import "github.com/metruzanca/gothic/browser"
```

## Functions

```go
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

browser.Navigate("/about")
browser.Replace("/about")
browser.Back()
```

| Function | Result |
|---|---|
| `Alert(message)` | Shows a browser alert |
| `Log(values...)` | Writes to the console |
| `SetTitle(title)` | Sets the document title |
| `GetItem(key)` | Reads `localStorage`, empty string when missing |
| `SetItem(key, value)` | Writes `localStorage` |
| `RemoveItem(key)` | Removes a `localStorage` key |
| `Copy(text)` | Writes text to the clipboard |
| `Focus(selector)` | Focuses the first matching element |
| `ClientHeight(selector)` | Reads an element height, or `0` |
| `Reload()` | Reloads the page |
| `Navigate(path)` | `pushState` and dispatch `LocationChanged` |
| `Replace(path)` | `replaceState` and dispatch `LocationChanged` |
| `Back()` | History back |

## Rules

- Calls are only valid in `Init`, `Update`, or a command. Using one in a view is
  a compile error.
- The Go stubs panic if they run on the server. The server never runs commands,
  so this is safe.
- `Navigate`, `Replace`, and `Back` require a `LocationChanged` message.
- The package does not expose arbitrary `window` access. Add a function here
  when an app genuinely needs a new browser API.

`ClientHeight` is useful for measurement commands:

```go
type Measure struct{}

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg.(type) {
	case Measure:
		return m, func() gothic.Msg {
			return Measured{Height: browser.ClientHeight("#rows")}
		}
	}
	return m, nil
}
```

## What next

The browser module covers small DOM and history calls. The next chapter covers
long lists with the separate `virtual` module:

- [Long lists](06-long-lists.md)
- [Syntax reference: browser interop](../SYNTAX.md#browser-interop)
