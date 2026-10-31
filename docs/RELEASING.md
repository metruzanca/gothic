# Releasing

Release tags are the published versions. Gothic and the UI are separate Go
modules with independent versions, so each gets its own tag:

- Gothic: `vX.Y.Z` (for example `v0.3.0`)
- UI: `ui/vX.Y.Z` (for example `ui/v0.1.0`)

Release the core first, then the UI. `ui/go.mod` requires a tagged core
version, so the UI tag must point at a commit made after the core tag exists.

Releases are initiated by the user only. Run the verification commands in
[AGENT.md](../AGENT.md) before tagging. Push `main` with each release, not only
the tag, so the tagged commits are reachable from the default branch.

## Core release

1. Confirm the test suite and generated files are current.
2. Tag the commit and push:

```sh
git tag -a vX.Y.Z -m vX.Y.Z
git push origin main
git push origin vX.Y.Z
```

The tag must not have a `ui/` prefix, and the version must be a valid semantic
version understood by Go, such as `v1.2.3` or `v0.4.0`.

## UI release

The UI is the `github.com/metruzanca/gothic/ui` module in `ui/`. Go requires
submodule tags to use the module's subdirectory as a prefix, so the tag is
`ui/vX.Y.Z`, never `vX.Y.Z`.

1. Remove `ui/go.work` in the release commit. It is a local development helper
   and must not ship in the tagged tree:

```sh
git rm ui/go.work
```

2. Bump the core requirement in `ui/go.mod` to the core release you are
   publishing against, then run `go mod tidy` in `ui/`:

```
require github.com/metruzanca/gothic vA.B.C
```

3. Update every example module that replaces the UI module and requires the
   core, currently `examples/gothicbook` and `examples/virtual`. Their
   `require github.com/metruzanca/gothic` line must match `vA.B.C`. Otherwise
   `go test` fails with `updates to go.mod needed`. Run `go mod tidy` in each.
   Check any other module that reports the same error.

4. Run the module checks:

```sh
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/virtual && go test ./...)
(cd examples/router && go test ./...)
```

5. Commit the version bump, tag, and push:

```sh
git add ui/go.mod ui/go.sum examples/gothicbook/go.mod examples/virtual/go.mod
git commit -m "chore(ui): release vX.Y.Z against gothic vA.B.C"
git tag -a ui/vX.Y.Z -m ui/vX.Y.Z
git push origin main
git push origin ui/vX.Y.Z
```

## Verify a release

Check that consumers can fetch both modules from an empty module cache:

```sh
cd "$(mktemp -d)"
go mod init example.com/releasecheck
go get github.com/metruzanca/gothic@vX.Y.Z
go get github.com/metruzanca/gothic/ui@vX.Y.Z
go list -m github.com/metruzanca/gothic github.com/metruzanca/gothic/ui
```

Module proxies cache repository refs for up to about 30 minutes. A newly pushed
tag can return 404 from `proxy.golang.org` even when it is valid. Verify it with
`GOPROXY=direct` and check `https://index.golang.org/index`, then retry the
proxy before changing anything.

Use the plain UI version in commands, for example
`go get github.com/metruzanca/gothic/ui@v0.1.0`. Go maps that to the
`ui/v0.1.0` git tag itself.

Do not move or delete a published tag. If a release is wrong, publish a new
patch version instead.
