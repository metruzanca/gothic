## Rules

- Use [Conventional Commits](https://www.conventionalcommits.org/).
- Releases are initiated by the user only.
- Use `templ` as the language identifier in Markdown code fences for gothic
  code. Do not use `gothic` or `gth` for now.

## Verification

Before committing, run:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
node --check client/runtime/gothic.js
node --check examples/pingpong/pingpong_gothic.js
node --check examples/router/app_gothic.js
```

For the UI, Gothicbook, and router modules, also run:

```sh
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
```

Regenerate `.gth` artifacts with `go generate ./examples/pingpong`,
`go run ./cmd/gothic generate ./ui`, or
`go run ./cmd/gothic generate ./examples/gothicbook` when source changes affect
the generated files. The router example regenerates from inside its own module
with `go generate ./...`.

Commit generated files (`*_gothic.go` and `*_gothic.js`) next to their
`.gth` source. Generated Go uses `//go:embed` for its sibling JavaScript, so
the generated files must exist at build time.
