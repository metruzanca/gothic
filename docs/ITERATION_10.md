# Iteration 10: actionable diagnostics

Status: complete. Parse and compiler errors now include the file, position,
source line, caret, and a hint.

## Goal

A failure from `go generate ./...` should tell you what is wrong and where to
look, without opening the compiler source.

## Before and after

Before:

```
gothic: app.gth:6:3: unexpected closing tag
```

After:

```
gothic: app.gth:7:3: unexpected closing tag
  7 |     </div>
           ^
  hint: the if block opened on line 5 is missing a closing }
```

A `js` call in a view:

```
gothic: app.gth:15:6: js.GetItem can only be used in Init, Update, or commands
  15 |         <p>{ js.GetItem("name") }</p>
                  ^
  hint: move js calls into Init, Update, or a command
```

A route on the wrong element:

```
gothic: app.gth:4:2: route is only supported on <a>
  4 |     <div route>
          ^
  hint: use <a href="/path" route> for client-side navigation
```

## Parse errors

`parser.Error` carries:

- file,
- line and column,
- message,
- source line,
- optional hint.

The parser builds errors through `errf`, `errh`, `withHint`, and `forceHint`.
The formatter expands tabs, prints the source line with a gutter, and places a
caret under the column.

Hints cover common structural mistakes:

- a missing closing brace for a view, `if`, or `for`,
- an unexpected closing tag,
- a mismatched component closing tag,
- a `route` attribute with a value,
- slot attributes or children.

## Real line numbers

`ParseApp` splits a `.gth` file into a Go prelude and a view, then parses a
synthetic view source with a leading package line. Without adjustment, view
positions were relative to that synthetic source.

The parser now carries a line offset and the original source. Node positions
and parse errors point at the real `.gth` lines. Hints that mention block
openings use the real lines too.

## View node positions

Every view node carries `NodePosition{File, Line, Col, Source}`:

- text,
- interpolation,
- elements,
- loops,
- conditionals,
- component calls,
- slots,
- templates.

Expansion copies positions, so errors from expanded pure components still
point at the source that defined the node.

## Generator errors

The Go generator and JavaScript compiler use node positions for:

- unsupported nodes,
- slots outside pure components,
- route links on non-anchor elements,
- route links outside the App view,
- missing `LocationChanged`,
- unknown components,
- duplicate, missing, or unknown props,
- stateful components nested where they are not supported.

`js` errors include the position of the surrounding view node. Unsupported
expressions and statements include their source text and a hint.

Update and command errors use the Go prelude file set, so they point at the
statement in the prelude.

## Go prelude errors

Invalid Go in the prelude now uses the same error format:

```
app.gth:3:1: invalid Go prelude: expected declaration
  3 | func broken( {
      ^
```

Missing app declarations are clearer:

```
app.gth: Model type not found; declare type Model struct{...}
app.gth: Model.Init not found; add func (m Model) Init() gothic.Cmd
app.gth: Model.Update not found; add func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd)
```

## CLI

- Parser errors are printed with their source context.
- If the modern app parse fails, the CLI no longer falls back to a worse legacy
  parse error when the failure is already a positioned view error.
- File prefixes are not duplicated when the error already carries a file.
- App validation errors point at the App or component view.
- Missing `Model`, `Init`, and `Update` report a hint.

## Tests

- parse errors include the original line, source excerpt, caret, and hint,
- `ParseApp` positions are rebased onto real lines,
- generator route errors carry source context,
- `js` errors in views carry source context,
- CLI failures report parse context and missing model hints.

## Verification

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
```

Manual checks:

```sh
go run ./cmd/gothic generate /path/to/broken-app
```

## Limits

- errors stop at the first failure; the CLI does not collect every file error,
- some internal generator errors still lack a hint,
- Go compiler errors after generation remain separate,
- no formatter, LSP, quick fixes, or source maps yet,
- hints are hand-written and need coverage as new checks are added.

## Follow-ups

- Collect multiple errors per run.
- Add hints for remaining generator checks.
- Formatter and LSP diagnostics.
- Source maps for generated JavaScript.
