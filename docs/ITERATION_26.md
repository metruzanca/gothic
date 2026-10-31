# Iteration 26: route tables

Status: complete. Routes are declared once and resolved on the server and in
the browser.

## Goal

Remove the duplicate path switches between `NewModel` and `Update`.

```go
var routes = []route.Route{
	{Pattern: "/", Page: "home"},
	{Pattern: "/about", Page: "about"},
	{Pattern: "/hello/:name", Page: "hello"},
}
```

## API

```go
match := route.Resolve(routes, path)
```

`route.Resolved` has:

- `Page`: the matched route's page,
- `Matched`: whether a route matched,
- `Params`: captured `:name` segments.

The first matching route wins. A miss returns `Matched: false` and an empty
params map.

Prefix a table for a nested section:

```go
admin := route.Join("/admin", routes)
```

`Join` returns a new table and does not mutate its input.

## Shared matching

- The server uses the real Go implementation.
- `route.Resolve` lowers to a `routeResolve` runtime helper.
- `route.Join` lowers to `routeJoin`.
- The table lives in the app model, so the client gets the same routes after
  hydration.

## Router example

The router example now has one route table:

```go
var routes = []route.Route{ ... }
```

`NewModel` and `Update` both call `route.Resolve(m.Routes, path)`. The hello
page reads `match.Params["name"]`. The previous duplicate path switch is gone.

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
(cd examples/nested && go test ./...)
(cd examples/virtual && go test ./...)
```

Tests cover first-match ordering, parameters, misses, prefixed tables, client
lowering, and the router example's deep links and module.

## Commits

The route table work ships in this iteration.

## Limits

- no wildcard or optional segments,
- no reverse URL builder,
- route tables are ordered,
- query and hash are not part of the pattern.

## Follow-ups

- Wildcard and optional segments.
- A reverse URL builder from a route and params.
- A generated route table shared by server and client without a model field.
