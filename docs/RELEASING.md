# Releasing

Release tags are the published versions. Gothic is split into four Go modules
with independent versions:

| Module | Path | Tag |
|---|---|---|
| Gothic | `github.com/metruzanca/gothic` | `vX.Y.Z` |
| UI kit | `github.com/metruzanca/gothic/ui` | `ui/vX.Y.Z` |
| Browser | `github.com/metruzanca/gothic/browser` | `browser/vX.Y.Z` |
| Virtual | `github.com/metruzanca/gothic/virtual` | `virtual/vX.Y.Z` |

Only release the modules that changed. Tags are never moved or deleted. If a
release is wrong, publish a new patch version.

Releases are initiated by the user only. Run the verification commands in
[AGENT.md](../AGENT.md) before tagging. Push `main` with each release, not only
the tag, so the tagged commits are reachable from the default branch.

## Dependency order

- `ui` requires a tagged core version.
- `browser` and `virtual` have no core dependency.
- `virtual/example` requires core and `virtual`.
- `examples/gothicbook` requires core, `ui`, and `browser`.
- `examples/router` requires core and `browser`.

Release core first when core changed. Then bump and tag the submodules that
depend on it. The example modules are development apps, not published modules;
bump their local requirements so the workspace builds.

## Core release

1. Confirm the test suite and generated files are current.
2. Tag the commit and push:

```sh
git tag -a vX.Y.Z -m vX.Y.Z
git push origin main
git push origin vX.Y.Z
```

The tag must not have a submodule prefix, and the version must be a valid
semantic version understood by Go, such as `v1.2.3` or `v0.4.0`.

## Submodule release

Each submodule tag uses the module directory as a prefix: `ui/vX.Y.Z`,
`browser/vX.Y.Z`, or `virtual/vX.Y.Z`, never a plain `vX.Y.Z`.

1. Bump the core requirement when the submodule depends on core:

```sh
cd ui
go mod edit -require=github.com/metruzanca/gothic@vA.B.C
go mod tidy
```

Repeat in any example module that requires core if you want the workspace to
build against the new core release.

2. Remove local workspace files from the release commit. They point at `..` and
   are only for development:

```sh
git rm ui/go.work
git rm browser/go.work
git rm virtual/go.work
```

3. Run the module checks:

```sh
(cd ui && go test ./...)
(cd browser && go test ./...)
(cd virtual && go test ./...)
(cd virtual/example && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
```

4. Commit the version bump, tag, and push:

```sh
git add ui/go.mod ui/go.sum
git commit -m "chore(ui): release vX.Y.Z against gothic vA.B.C"
git tag -a ui/vX.Y.Z -m ui/vX.Y.Z
git push origin main
git push origin ui/vX.Y.Z
```

## Verify a release

Check that consumers can fetch the modules from an empty module cache:

```sh
cd "$(mktemp -d)"
go mod init example.com/releasecheck
go get github.com/metruzanca/gothic@vX.Y.Z
go get github.com/metruzanca/gothic/ui@vX.Y.Z
go get github.com/metruzanca/gothic/browser@vX.Y.Z
go get github.com/metruzanca/gothic/virtual@vX.Y.Z
go list -m github.com/metruzanca/gothic github.com/metruzanca/gothic/ui github.com/metruzanca/gothic/browser github.com/metruzanca/gothic/virtual
```

Module proxies cache repository refs for up to about 30 minutes. A newly pushed
tag can return 404 from `proxy.golang.org` even when it is valid. Verify it with
`GOPROXY=direct` and check `https://index.golang.org/index`, then retry the
proxy before changing anything.

Use the plain submodule version in commands, for example
`go get github.com/metruzanca/gothic/ui@v0.1.0`. Go maps that to the
`ui/v0.1.0` git tag itself.
