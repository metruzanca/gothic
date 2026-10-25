# Agent guidelines

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/).

Format:

```
<type>(<optional scope>): <summary>
```

- Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`,
  `ci`, `chore`, `revert`.
- Use the imperative mood in the summary.
- Keep the summary lower-case and do not end it with a period.
- Keep each commit focused. Do not mix unrelated changes.
- Mark breaking changes with `!` after the type or scope and explain them in a
  `BREAKING CHANGE:` footer.

Examples:

```
feat(parser): support nested for loops
fix(jsgen): avoid duplicate module script tags
refactor(syntax)!: call rpc procedures directly
docs: add syntax reference
```

## Releases

Releases are initiated by the user only.

Do not create git tags, GitHub releases, version bumps, or changelogs unless the
user explicitly asks for a release. Committing changes is not a request to
release.

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

Regenerate `.gothic` artifacts with `go generate ./examples/pingpong` or
`go run ./cmd/gothic generate .` when source changes affect the generated files.

## Generated files

Commit generated files (`*_gothic.go` and `*_gothic.js`) next to their
`.gothic` source. Generated Go uses `//go:embed` for its sibling JavaScript, so
the generated files must exist at build time.
