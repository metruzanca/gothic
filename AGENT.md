## Rules

- Use [Conventional Commits](https://www.conventionalcommits.org/).
- Releases are initiated by the user only.

## Verification

Before committing, run:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
node --check client/runtime/gothic.js
node --check examples/pingpong/views/pingpong_gothic.js
```

Regenerate `.gth` artifacts with `go generate ./examples/pingpong` or
`go run ./cmd/gothic generate .` when source changes affect the generated files.

Commit generated files (`*_gothic.go` and `*_gothic.js`) next to their
`.gth` source. Generated Go uses `//go:embed` for its sibling JavaScript, so
the generated files must exist at build time.
