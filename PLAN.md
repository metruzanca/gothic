# gothic: a fullstack Go framework that compiles templ-like components to Go and JavaScript

> Status: iteration 1 is implemented and verified. See [ITERATION_1.md](ITERATION_1.md).
> Iteration 1 uses a bootstrap compiler; the templ hard fork migration is iteration 2.

## Overview

gothic is a hard fork of [templ](https://github.com/a-h/templ) that extends templ's
component language so that a single `.gothic` source file compiles to both:

- Go server code that renders HTML (SSR), and
- JavaScript that performs fine-grained DOM updates in the browser (no virtual DOM).

The goal is to fill the biggest gap in HTMX-style workflows: today you must leave the
server language and write JavaScript (or bolt on Alpine.js) as soon as you need real
client-side state, bindings, or events. gothic makes that interactivity a first-class,
compiled part of the component language, while keeping server rendering and progressive
enhancement.

The target pipeline is:

```
.gothic source
    |
gothic generate
    |-- _gothic.go     (SSR component, hydration markers, serialized state)
    |-- _gothic.js     (hydrate()/create() plus binding descriptors)
    |-- _gothic.json   (manifest: exports, hydration roots, state schema)
    |
go build / go run
    |-- serves HTML and assets
    |-- embeds the JS runtime and generated modules via go:embed
```

No npm is required on the happy path. Generated JavaScript ships as ES modules served
from the Go binary.

## Decisions locked

| Area | Decision |
|---|---|
| Expression semantics | Transpile a documented subset of Go expressions to JavaScript. Reuse templ's `go/parser` based expression parsing. |
| Reactivity model | Runtime signals with a deep proxy store and auto-tracking. Effects patch bound DOM nodes. No virtual DOM. |
| SSR / hydration | Full hydration with markers. The Go backend emits markers; the JS backend adopts the existing DOM in place. |
| Fork strategy | Hard fork of the whole templ repository into gothic. Grammar extensions land in the forked parser. |
| Default router | chi v5 as the blessed default over an `net/http` `http.Handler` core. |
| Internal RPC | gothic-native generated JSON procedures with a no-JS form-POST fallback. |
| External API | REST + OpenAPI 3.1 via huma, mounted alongside page routes and procedures. |

## Syntax direction

The embedded code is Go, not JavaScript. The parser uses `go/parser` for state
initializers, loop headers, interpolations, and event handler bodies.

- State is declared as `state name = <Go expression>`.
- Lists use `for ... range`, with templ-style markup in the body.
- Event handlers are Go function literals: `onclick={ func() { ... } }`.
- Procedures are declared with `rpc Name() string`.
- Procedure calls are awaited automatically. There is no `await` keyword in the
  source.
- Two-value procedure calls use Go error handling: `v, err := api.Proc()`.
- `append` on state lowers to a reactive push.
- `nil` and `err.Error()` lower to their JavaScript equivalents.

There is no state, `@each`, `.Push`, or JavaScript array literal in the source
language. See [SYNTAX.md](SYNTAX.md) for the current grammar and limits.

## Repository layout

```
gothic/
  cmd/gothic/                     # CLI: generate, fmt, lsp, proxy (renamed from cmd/templ)
  parser/v2/                      # forked AST + parser (grammar extensions land here)
  generator/                      # existing Go generator, kept close to upstream
  generator/jsgen/                # NEW: JS backend, sibling walker over the same AST
  expr/js/                        # NEW: Go-expression to JS translator
  runtime/                        # existing Go runtime (markers, state serialization)
  gothic/                         # root package (renamed from templ)
  server/                         # NEW: router wrapper, chi default, Mount escape hatch
  server/routes/                  # NEW: route registry + generated typed URL helpers
  server/rpc/                     # NEW: procedure runtime (encoding, errors, CSRF, fallback)
  server/openapi/                 # NEW: huma integration (optional import)
  client/runtime/                 # NEW: gothic.js signals + DOM helpers (embedded)
  client/build/                   # NEW: manifest, hashing, go:embed glue
```

Generated artifacts:

```
foo.gothic -> foo_gothic.go       # SSR component + state + hydration markers
           -> foo_gothic.js       # hydrate() + create() + binding descriptors
           -> foo_gothic.json     # manifest entry for this file
```

## Architecture

### Shared AST, two walkers

`parser/v2` remains the single source of truth. The existing Go generator is kept close
to upstream; it is not parameterized. Instead, `generator/jsgen` is a sibling walker over
the same `*parser.TemplateFile`, mirroring templ's `writeNode` dispatch. A shared
hydration-marker scheme is emitted by both backends from the start so their outputs stay
structurally comparable.

### JS output shape

Each template compiles to an ES module exporting `create(parent, props, state)` for
dynamic insertion and `hydrate(node, state)` for adopting server HTML. Static subtrees
are created once and never touched. Only bound slots register effects that patch the DOM.

### Client runtime (`gothic.js`, embedded)

- `signal(v)`: callable getter/setter with dependency tracking.
- `effect`, `computed`, microtask batching, untracked reads.
- `store(obj)`: deep reactive proxy so `user.Name` auto-tracks without explicit
  dependency lists. Exported-name casing is preserved from Go so the same expression text
  works verbatim in both backends.
- DOM helpers: `el`, `text`, `attr`, `on`, `each` (keyed), `when`, `bind` (two-way).
- Hydration reader that walks markers, binds effects to real nodes, then removes markers.

### Expression transpilation (`expr/js`)

Input is the `go/parser` AST of `Expression.Value` (templ already parses expressions this
way). Output is JavaScript. The supported subset is documented and enforced:

- Identifiers, selectors, index/slice, parens, literals, unary/binary operators.
- `&&`, `||`, `!` map directly. Equality and conversion rules are defined explicitly.
- Calls resolve through a whitelist (`len` to `.length`, `string`/`int` conversions, and
  so on), through client-callable functions, or fail with a positioned compile error.
- Go statements in `if`/`switch`/`for` nodes are parsed and translated (`if` to `if`,
  `for ... range` to `for (const x of ...)`, `:=` to `let`).
- An escape hatch (`gothic.JSExpr("...")`) allows raw JS when the subset is insufficient.

The `go/packages` annotation pass (see Phase 0.5) provides real type information, which
lets the transpiler resolve method calls and composite values instead of guessing from
syntax. This substantially narrows the "syntactically unambiguous subset" restriction.

## Language extensions

templ's primitives (if/switch/for, elements, components, children) are reused largely
as-is for SSR. The reactive surface is new:

- State declarations per component.
- Event bindings that compile to handlers rather than string attributes.
- Reactive conditionals and lists, distinct from server `if`/`for`.
- Two-way input binding.
- Client-callable functions (an evolution of templ's `script` templates and `JSFuncCall`).
- An island / client-component boundary directive.
- An async resource primitive for fetch and SSE.

## Routing, procedures, and API

### Router

`net/http` is the core. `gothic.NewServer()` returns the blessed default (chi v5 plus
procedures plus the route registry). `gothic.Mount(mux, ...)` attaches everything to any
user mux, so there is no lock-in. The reserved prefix `/_gothic/` is used for framework
endpoints to avoid collisions with user routes.

### Route-awareness

Routes are declared in Go with a marker comment (`//gothic:route`) and resolved by the
`go/packages` pass. `gothic generate` emits `routes.*` symbols and typed path builders,
so references from `.gothic` files break at build time when a route is renamed or
removed.

### Internal procedures

Procedures are declared as Go functions with a marker comment so both backends receive
real types:

```go
//gothic:procedure
func CreateUser(ctx context.Context, in UserInput) (User, error) { ... }
```

`gothic generate` emits a Go handler registration at `POST /_gothic/rpc/CreateUser`, a
typed JS fetch stub for `.gothic` expressions such as `api.CreateUser(input)`, and a
manifest entry with input/output schemas.

Behavior:

- JSON request/response with a typed error envelope mapped to HTTP status.
- Same-origin session cookie plus a CSRF token (`X-Gothic-CSRF`).
- Internal procedures are not reachable cross-origin by default.
- Progressive enhancement: the same endpoint accepts form-encoded input and returns an
  HTML fragment (a `.gothic` component) when `Accept: text/html`, so plain forms work
  without JavaScript.
- Errors align to RFC 9457 problem+json so internal and external consumers share one
  contract.
- Validation uses a single source (go-playground/validator struct tags), shared with huma
  to prevent drift.
- SSE is the extension point for subscriptions and async resources, added later without
  changing the unary contract.

### External API

huma generates OpenAPI 3.1 from typed handlers on the same chi router. External routes are
declared explicitly and never auto-derived from internal procedures, so a new procedure
cannot accidentally become public. Security schemes (API key, OAuth) live in OpenAPI
metadata and stay separate from internal cookie sessions.

## Phases

### Phase 0: Fork and rename

Mechanical but wide-reaching. Clone templ, re-init git, then:

- Rename module path and root/runtime package identifiers.
- Rename the CLI binary from `templ` to `gothic`.
- Change the source extension to `.gothic` and generated suffix to `_gothic.go`.
- Update generated-file markers and strip unneeded upstream release/CI surface.

Acceptance: `go build ./...`, `go test ./...`, and `go run ./cmd/gothic generate` on a
sample `.gothic` file all pass. Keep the rename in one isolated commit so upstream merges
stay feasible.

### Phase 0.5: Go annotation scanning

Add a `go/packages` pass to `generatecmd` that finds routes and procedures and extracts
type information. Cache results and run in parallel with `.gothic` parsing. This pass
feeds route/procedure codegen and gives the expression transpiler its type information.

Acceptance: markers are discovered and typed, incremental and repeat generates are fast.

### Phase 1: JS backend as a sibling walker

Add `generator/jsgen` over the same AST. First subset, no reactivity: elements, constant
and expression attributes, text, doctype, children, component calls, string expressions.
Emit `create(parent, props)`.

Acceptance: a static nested template renders byte-equivalent DOM in Go and in the browser
(compare with chromedp against golden HTML, reusing templ's harness).

### Phase 2: Expression transpiler

Build `expr/js`. Reuse templ's expression parser, translate the documented subset, and
consume type information from Phase 0.5 where available.

Acceptance: golden matrix of expressions; unsupported input yields a clear, positioned
error and never silently wrong JavaScript.

### Phase 3: Reactive runtime and language extensions

Build `gothic.js` (signals, proxy store, effects, DOM helpers, two-way binding). Extend
the grammar for state, event bindings, reactive conditionals/lists, client functions, and
islands. The server serializes initial state safely, reusing templ's JSON escaping.

Acceptance: counter, todo, and async fetch demos run in-browser with no virtual DOM and no
npm, with correct SSR.

### Phase 3.5: Procedures and route codegen

Build `server/`, `server/routes/`, and `server/rpc/`. Generate Go handlers, typed JS
stubs, the route registry, and the procedure manifest. Implement CSRF, the error envelope,
and the form-POST HTML fallback.

Acceptance: a `.gothic` component calls a procedure from the browser and via a no-JS form
POST, both returning correct results, with route references compile-checked.

### Phase 4: Full hydration

Define the marker scheme: comment nodes for ranges and a `data-gothic` attribute for bound
elements (stripped during hydration). The Go emitter emits markers; the JS emitter bakes
matching IDs into binding descriptors. `hydrate(node, state)` adopts the DOM, binds
effects, and removes leftover markers.

Acceptance: server HTML and post-hydration DOM are identical except marker removal, with
no flash, and a reflective test asserts one effect per dynamic slot.

### Phase 5: Serving glue

Build `gothic.NewServer`, `gothic.Mount`, and asset mounting via `go:embed` with content
hashing, immutable caching, and an import map or single bundled ESM entry. Wire huma on
the same router. Keep the build npm-free, with watch mode reusing templ's watcher and SSE
reload.

Acceptance: `go build && ./app` serves a working hydrated app plus OpenAPI docs from one
binary.

### Phase 6: Developer experience

Formatter support for the new syntax, LSP completion for state, event bindings, client
functions, routes, and procedures. JS source maps back to `.gothic`, extending templ's
sourcemap work. Docs and a `gothic new` scaffold.

### Milestone 1 (walking skeleton)

Before any phase is considered done, prove the whole pipeline end to end: a single
component with one state variable and one click handler, SSR'd by Go and hydrated by
`gothic.js`, served from one `go build` binary with zero npm. This is the falsifiable
version of the thesis and should be the first thing built after Phase 0.

## Risks

- Expression transpilation is the main correctness risk. Type information from Phase 0.5
  mitigates it, but the subset remains a permanent constraint on the language.
- Deep-proxy tracking and Go-style method calls do not perfectly align. Define the
  client-callable function namespace early.
- Full hydration doubles the work on the Go emitter. This is why markers are introduced in
  Phase 1, long before they are needed.
- The `go/packages` pass requires the module to compile and adds latency. It must be
  incremental and cacheable, or generate will feel slow.
- One validation source only. Splitting validation between procedures and huma will drift.
- Default-deny external exposure of procedures must be enforced loudly in codegen, not
  only documented.
- Fork drift. Keep upstream changes mergeable by confining edits to `parser/v2` additions
  and new packages.
- Do not make protobuf or ConnectRPC mandatory. Keep them as an optional adapter if
  streaming or polyglot consumers arrive later.

## Open questions

- Confirm the final source extension (`.gothic` assumed).
- Confirm procedure/route declaration as Go marker comments (recommended, gives real
  types) versus a `procedure` block inside `.gothic` files.
- Decide the exact equality and numeric-conversion rules for the expression subset.
- Decide when ConnectRPC and SSE adapters enter the roadmap.

## Prior art

- **ghtmx**: a hard fork of templ that adds compile-checked htmx, proving fork viability
  and syntax retention.
- **ily.dev/domi**: Go server framework syncing the DOM with a tiny `client.js`, no VDOM.
- **tinywasm/dom**: fine-grained signal-based DOM reactivity in Go.
- **Gea**, **Svelte**, **Solid**: compile-time surgical DOM updates; Svelte's hydration
  markers are directly relevant.
- **huma**, **Fuego**, **chi**: OpenAPI and routing on `net/http`.
- **ConnectRPC**: optional polyglot/streaming RPC adapter.

## Iteration 1 status

Iteration 1 is implemented and verified. The detailed record is in
[ITERATION_1.md](ITERATION_1.md).

Delivered:

- A `.gothic` parser for the iteration 1 subset.
- A Go SSR generator with hydration markers and state serialization.
- A JavaScript generator for signals, list adoption, event binding, and RPC calls.
- An embedded client runtime with signals, effects, list hydration, and a JSON
  procedure transport.
- A chi-based server that serves the runtime, generated modules, and procedures.
- A `gothic generate` CLI.
- The ping/pong example, built from Go and `.gothic` only, with no hand-written
  JavaScript and no wasm.
- Unit and integration tests, plus browser verification.

Deliberate deviation: iteration 1 uses a clean-room bootstrap compiler instead of a
completed templ hard fork. The slice needs only a tiny grammar, and a partially
modified templ fork would add coupling and merge risk before the pipeline is proven.
The emitted contract (markers, module URL, runtime API, server routes, procedure
registrars) is designed to survive the fork migration. Iteration 2 replaces the
bootstrap parser and generators with the templ fork while keeping that contract stable.

Accepted iteration 1 limits:

- Static root id, one instance per page.
- Unkeyed list, string-slice state, zero-argument procedures.
- No events inside lists, nested lists, or top-level dynamic expressions.
- No CSRF, auth, or no-JS form fallback.
- The component emits its own module script tag.
- No bundling or minification.
