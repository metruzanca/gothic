## Rules

- Use [Conventional Commits](https://www.conventionalcommits.org/).
- Releases are initiated by the user only.
- Use `templ` as the language identifier in Markdown code fences for gothic
  code. Do not use `gothic` or `gth` for now.

## No legacy paths

- There is one supported way to do each thing. Do not keep old syntax, APIs, or
  configuration for compatibility.
- When a feature changes, migrate every call site, example, test, and doc in the
  same change, then delete the old code path. Do not add shims, deprecation
  notes, or "older X still works" wording.
- Example: components take a props struct. Individual-prop components are not
  supported and must not be reintroduced.

## Verification

Before committing, run:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
node --check client/runtime/gothic.js
node --check examples/guestbook/app_gothic.js
node --check examples/router/app_gothic.js
node --check virtual/example/app_gothic.js
```

For the nested modules, also run:

```sh
(cd ui && go test ./...)
(cd browser && go test ./...)
(cd virtual && go test ./...)
(cd virtual/example && go test ./...)
(cd session && go test ./...)
(cd auth && go test ./...)
(cd examples/accounts && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
```

Regenerate `.gth` artifacts with `go generate ./examples/guestbook`,
`go run ./cmd/gothic generate ./ui`, or
`go run ./cmd/gothic generate ./examples/gothicbook` when source changes affect
the generated files. The router example regenerates from inside its own module
with `go generate ./...`.

Commit generated files (`*_gothic.go` and `*_gothic.js`) next to their
`.gth` source. Generated Go uses `//go:embed` for its sibling JavaScript, so
the generated files must exist at build time.

## Releases

Releases are initiated by the user only. See
[docs/RELEASING.md](docs/RELEASING.md) for the module release process.
