# Iteration 16: path parameters

Status: complete. The `route` package matches path patterns and captures named
segments on the server and in the browser.

## Goal

Stop writing prefix checks and string splitting by hand for routes like
`/todos/42`. Match a pattern, capture the parameter, and use it in the model.

## API

```go
import "github.com/metruzanca/gothic/route"

params := route.Params("/todos/:id", msg.Path)
if params != nil {
	m.ID = params["id"]
}

m.Edited = route.Match("/todos/:id/edit", msg.Path)
```

- `route.Params(pattern, path)` returns `map[string]string`, or `nil` when the
  path does not match.
- `route.Match(pattern, path)` returns the boolean result.
- A segment that starts with `:` captures one non-empty segment.
- The pattern and path must have the same number of segments.
- Captured segments are percent-decoded on both sides.

## Server and client

The server imports the real Go package:

```go
params := route.Params("/hello/:name", path)
```

The compiler lowers the same call in `Update` to a runtime helper:

```js
const params = routeParams("/hello/:name", msg.Path);
if (params != null) {
  m.Name = params["name"];
}
```

Both implementations use the same segment rules, so SSR and hydration agree.
`route.Params` returns `null` in JavaScript and `nil` in Go for a miss, which
means the existing nil checks work unchanged.

This reuses the collection work from Iteration 15: the result is a map, so
`params != nil`, `len(params)`, indexing, and range loops all work.

## If init support

The compiler now emits an `if` init statement before the condition:

```go
if params := route.Params("/todos/:id", msg.Path); params != nil {
	m.ID = params["id"]
}
```

Before, the init statement was silently dropped. It now appears in the
generated JavaScript.

## Router example

`examples/router` gained a `/hello/:name` page:

- `NewModel` matches the path on the server,
- `Update` matches the path after client navigation,
- the view renders the captured name,
- `/hello/` still falls through to the not-found page.

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

- the Hello nav link navigates to `/hello/Ada` without a reload,
- the heading reads `Hello, Ada`,
- the Hello link becomes active,
- no not-found branch appears.

Tests cover the Go matcher, the JavaScript lowering, if init emission, server
deep links, and unknown route functions.

## Commit

- `4833c52` path patterns and parameters

## Limits

- no wildcard or optional segments,
- no trailing-slash normalization,
- duplicate parameter names keep the last segment,
- query and hash are not part of the pattern,
- apps still match once on the server and once in `Update`, because arbitrary
  prelude functions are not compiled to JavaScript,
- no declared route table or reverse URL builder.

## Follow-ups

- Nested routers and wildcard or optional segments.
- A declared route table that generates both the server and client matchers.
- Nested stateful components.
- The virtualized list example.
