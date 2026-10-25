# gothic syntax

gothic keeps the templ-style markup and makes the embedded code Go rather than
JavaScript. State initializers, event handler bodies, loop headers, and
interpolations are parsed with Go's own parser. There is no `await`, no
JavaScript array literal, and no custom statement dialect.

## Example

```gothic
package pingpong

templ PingPong() {
	state messages = []string{"ready"}

	<div class="app">
		<button onclick={ func() {
			messages = append(messages, "ping")
			pong, err := api.Ping()
			if err != nil {
				messages = append(messages, err.Error())
				return
			}
			messages = append(messages, pong)
		} }>Ping</button>
		<ul>
			for _, m := range messages {
				<li>{ m }</li>
			}
		</ul>
	</div>
}

rpc Ping() string
```

## Declarations

### Package

```gothic
package pingpong
```

The package name is used for the generated Go file.

### Template

```gothic
templ Name() {
	...
}
```

A template must have exactly one root element. Template parameters are parsed
but not yet compiled into the client backend.

### State

```gothic
state name = <Go expression>
```

One state declaration per line, at the top of the template. The expression is
parsed as Go. The current runtime supports string slices, for example
`[]string{"ready"}`. The generated Go uses `name := <expression>`. The generated
JavaScript creates a signal from the translated expression.

### Procedure

```gothic
rpc Ping() string
```

Zero-argument procedures only in this iteration. The generated Go interface is:

```go
type <Template>Procedures interface {
	Ping(ctx context.Context) (string, error)
}
```

The generated JavaScript exposes `api.Ping()`.

## Markup

Elements, constant attributes, text, and interpolation work as in templ:

```gothic
<div class="app">
	<p>Hello</p>
</div>
```

Interpolation takes a Go expression:

```gothic
<li>{ m }</li>
```

The expression is parsed with `go/parser`. In the current backend it must be an
identifier or another supported expression.

## Lists

```gothic
for _, m := range messages {
	<li>{ m }</li>
}
```

The range source must be a state identifier in this iteration. The Go backend
emits a normal range loop. The JavaScript backend emits a reactive list binding
and adopts the server-rendered children.

Go range semantics apply:

- `for k, v := range xs` gives index and value.
- `for k := range xs` gives the index.
- `for range xs` gives neither.

## Event handlers

```gothic
onclick={ func() {
	...
} }
```

The handler value is a Go function literal with no parameters and no results.
The generated JavaScript handler is async. Procedure calls are awaited
automatically, so the source never writes `await`.

Supported statements in this iteration:

- `x := expr`
- `x, err := api.Procedure(...)`
- `state = append(state, ...)`
- `state = expr`
- `if cond { ... }` with an optional `else` block
- `return` and `return expr`
- expression statements, including `api.Procedure(...)`

Supported expressions:

- identifiers, selectors, calls
- string, integer, float, and bool literals
- slice composite literals such as `[]string{"a", "b"}`
- binary and unary operators
- indexing and slicing
- `nil`
- `err.Error()`

## Semantics

### Procedure calls

`api.Ping()` compiles to `await api.Ping()` inside an async handler. Calling a
procedure outside an event handler is a compile error.

`pong, err := api.Ping()` compiles to a try/catch. The error variable receives
the rejection, so Go-style error handling works:

```gothic
pong, err := api.Ping()
if err != nil {
	messages = append(messages, err.Error())
	return
}
```

`err.Error()` compiles to `err.message`, and `err != nil` compiles to
`err != null`.

### State updates

`messages = append(messages, x)` compiles to a signal push.

Any other assignment to a state variable compiles to a signal set.

In-place mutation such as `messages[0] = x` is not supported yet.

### Lists

A `for ... range` over a state identifier compiles to a reactive list. The
server renders the initial children. The client adopts them and appends or
removes nodes as the signal changes.

## Current limits

- One state declaration per line.
- String-slice state only.
- Zero-argument procedures only.
- Range source must be a state identifier.
- One element root per template and per list iteration.
- No events inside lists.
- No nested lists.
- No top-level interpolations outside lists.
- No expression-valued attributes other than event handlers.
- No CSRF, auth, or no-JS form fallback.
- Static root id and one instance per page.
