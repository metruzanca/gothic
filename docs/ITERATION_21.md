# Iteration 21: documents and social previews

Status: complete. `MountApp` renders a full HTML document with sane metadata,
icons, and a framework social card. Gothicbook adopts it and deploys from the
in-repo modules.

## Goal

Sharing a link to a Gothic app should produce a usable preview without any
setup: a page title, a description slot, a favicon, and an OpenGraph image.
Apps that want their own branding override the defaults.

## Why the document lives in Go

The `.gth` app view compiles to both server HTML and client JavaScript, and its
output is the hydratable root inside `<body>`. The head must be static server
HTML because social crawlers do not run JavaScript. So the document is built in
Go, from the app entrypoint, not in `app.gth`.

## Default document

`MountApp` now writes:

```html
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Gothic App</title>
<link rel="icon" href="/_gothic/icon.svg" type="image/svg+xml">
<link rel="icon" href="/_gothic/icon-512.png" type="image/png" sizes="512x512">
<link rel="apple-touch-icon" href="/_gothic/apple-touch-icon.png" type="image/png" sizes="180x180">
<meta property="og:type" content="website">
<meta property="og:title" content="Gothic App">
<meta property="og:site_name" content="Gothic App">
<meta property="og:url" content="https://example.com/about">
<meta property="og:image" content="https://example.com/_gothic/og.png">
<meta name="twitter:card" content="summary_large_image">
</head>
<body>
...
</body>
</html>
```

Defaults:

- `lang="en"`
- title `Gothic App`
- `og:type` `website`
- `og:site_name` falls back to the title
- `og:url` is the absolute request path, without the query string
- `og:image` is the framework card below
- `twitter:card` is `summary_large_image` when an image exists, else `summary`
- relative `og:image` and `og:url` values become absolute using
  `X-Forwarded-Proto`, `X-Forwarded-Host`, `r.TLS`, and `r.Host`

A render failure is buffered and becomes a clean 500 with no partial document.

## Overrides

Server-wide defaults apply to every `MountApp` on that server:

```go
s := gothic.NewServer()
s.SetDocument(gothic.Document{
	SiteName: "Gothic",
	Styles:   []string{"/gothic-ui.css", "/style.css"},
})
```

Per-app options start from the server document:

```go
s.MountApp("/", gothic.NewApp(model),
	gothic.WithTitle("Gothicbook"),
	gothic.WithDescription("A tiny design system demo for the gothic UI kit."),
	gothic.WithImage("/og.png"),
	gothic.WithStyles("/gothic-ui.css", "/style.css"),
)
```

Available options:

- `WithDocument` replaces the configuration, subject to defaults.
- `WithDocumentFunc` merges a document chosen from the request. Non-zero
  fields and non-nil slices win. Use it for per-route titles.
- `WithTitle`, `WithDescription`, `WithSiteName`, `WithLanguage`,
  `WithThemeColor`
- `WithIcon` replaces the icon set with one icon whose MIME type is inferred.
- `WithIcons` sets an explicit set.
- `WithStyles` replaces the stylesheet list.
- `WithImage` sets `og:image`. An empty value suppresses it.
- `WithNoImage` suppresses the framework card and emits `summary`.
- `WithExtraHead` appends trusted markup.

`Document` is also usable without a Model:

```go
gothic.WriteDocument(w, r, gothic.Document{Title: "Status"}, body)
```

## Assets

The framework serves these from core under the reserved `/_gothic/` prefix:

| Path | Content | Source |
| --- | --- | --- |
| `/_gothic/icon.svg` | favicon | `assets/gothic-mark.svg` |
| `/_gothic/icon-512.png` | PNG favicon fallback | `assets/icons.html` |
| `/_gothic/apple-touch-icon.png` | iOS home screen | `assets/icons.html?background=opaque` |
| `/_gothic/og.png` | social card, 1200x630 | `assets/og.html` |

All four are `//go:embed` values and are served with
`Cache-Control: public, max-age=3600`. Apps can override the metadata, but not
the bytes at those URLs, because `/_gothic/` is handled before the router. Serve
a custom card at your own path, such as `/og.png`, and point `WithImage` at it.

### Regenerating the PNGs

The PNGs are committed. Regenerate them with headless Chrome:

```sh
cd assets
USER_DATA=$(mktemp -d)
google-chrome --headless=new --disable-gpu --hide-scrollbars \
  --no-sandbox --user-data-dir="$USER_DATA" \
  --default-background-color=00000000 \
  --window-size=512,512 --screenshot=icon-512.png \
  "file://$PWD/icons.html"
google-chrome --headless=new --disable-gpu --hide-scrollbars \
  --no-sandbox --user-data-dir="$USER_DATA" \
  --default-background-color=00000000 \
  --window-size=180,180 --screenshot=apple-touch-icon.png \
  "file://$PWD/icons.html?background=opaque"
google-chrome --headless=new --disable-gpu --hide-scrollbars \
  --no-sandbox --user-data-dir="$USER_DATA" \
  --default-background-color=00000000 \
  --window-size=1200,630 --screenshot=og.png \
  "file://$PWD/og.html"
```

`icon-512.png` is RGBA with a transparent background. The touch icon is opaque
`#f7f7f5`, because iOS fills transparency with black.

## Gothicbook

The example passes its own metadata and a custom card:

```go
s.MountApp("/", gothic.NewAppFunc(...),
	gothic.WithTitle("Gothicbook"),
	gothic.WithSiteName("Gothic"),
	gothic.WithDescription("A tiny design system demo for the installable gothic UI kit."),
	gothic.WithImage("/og.png"),
	gothic.WithStyles("/gothic-ui.css", "/style.css"),
)
```

`examples/gothicbook/assets/og.png` is a 1200x630 card rendered from
`assets/og.html` with the same headless Chrome flow. The example's `go.mod`
uses local replaces so `go test` runs against this checkout. The Railway build
drops them and fetches the core and UI modules at the deployed commit through
`examples/gothicbook/railpack.json`.

## Tests

`document_test.go` covers defaults, options, escaping, forwarded headers,
absolute image passthrough, image suppression, per-request documents, and error
propagation. `server_test.go` covers asset content types and caching, mount
options, and a clean 500 for a failing view. The examples that already used
`MountApp` keep passing because they assert substrings, not whole documents.

## Limits

- Safari on iOS wants a PNG `apple-touch-icon`, which the defaults provide.
  Assuming a configured SVG icon is enough for every platform is not safe.
- `/_gothic/` is reserved, so an app cannot override the framework assets in
  place.
- `HEAD` requests are still not registered by `Server.Get`, unchanged from
  before this iteration.
- The gothicbook demo tracks the in-repo modules, so a broken main reaches the
  demo. Released module versions still gate real consumers.
