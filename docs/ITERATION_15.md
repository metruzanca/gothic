# Iteration 15: broader client Go subset

Status: complete. `Update` and commands accept more collection expressions,
range loops, and non-type switch statements.

## Goal

Let App and component logic stay in Go for common list and map work instead of
falling back to `js` interop or duplicating logic in views.

## Expressions

Supported in `Update`, `Init`, constructors, commands, and view expressions:

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

- Indexing: `x[i]`.
- Two-index slicing: `x[lo:hi]`, `x[:hi]`, and `x[lo:]`.
- Map literals with computed keys.
- `len(x)` for slices, arrays, strings, and maps.
- `delete(map, key)`.
- Compound assignment: `+=`, `-=`, `*=`, `/=`, `%=`, `&=`, `|=`, `^=`, `<<=`,
  and `>>=`.
- `++` and `--` statements.

Three-index slices and `&^=` are rejected with source errors.

`len` moved to a runtime helper so map literals count keys:

```js
export function len(value) {
  if (value == null) return 0;
  if (Array.isArray(value) || typeof value === "string") return value.length;
  if (typeof value === "object") return Object.keys(value).length;
  return 0;
}
```

## Range loops

```go
for i, item := range m.Items {
	if item == "" {
		continue
	}
	m.Items[i] = item
}

for _, count := range m.Counts {
	m.Total += count
}
```

Generated loops use a runtime `entries` helper so the same lowering works for
slices and maps:

```js
for (const [i, item] of entries(m.Items)) {
```

`break` and `continue` are supported. Labeled branches and `fallthrough` are
rejected with source errors.

## Switch

Non-type switches compile to JavaScript switches. Go does not fall through, so
each clause gets an explicit `break`:

```go
switch msg.Path {
case "/":
	m.Page = "home"
case "/about":
	m.Page = "about"
default:
	m.Page = "notfound"
}
```

Tagless switches compile to `switch (true)`. A switch init statement is emitted
before the switch. The router example now maps paths with a switch instead of an
if chain.

## Semantic gaps

These differences are documented rather than fixed:

- A missing map key reads as `undefined`, not the Go zero value. Use a fallback
  when the value can be absent.
- Map iteration order is JavaScript insertion order, not Go's randomized order.
- Map keys arrive as strings, so an `int` map key becomes a string.
- `/=` is floating division in JavaScript.
- Slices copy in JavaScript, while Go slices share a backing array.
- `len(nil)` is 0 on both sides.

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

JavaScript modules pass `node --check`. Compiler tests cover indexing, slicing,
map literals, `len`, `delete`, compound assignment, `++`/`--`, range loops with
`break` and `continue`, and switch statements, plus rejection of three-index
slices, `&^=`, and `fallthrough`.

## Commits

- `0bdc870` collection expressions
- `0210784` range loops and switch statements

## Limits

- no labeled `break` or `continue`,
- no `goto` or `fallthrough`,
- no three-index slices,
- no map zero-value reads,
- view expressions still cannot declare variables or call procedures.

## Follow-ups

- Path parameters and route matching helpers.
- Nested stateful components.
- The virtualized list example.
