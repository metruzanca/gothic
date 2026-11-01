module github.com/metruzanca/gothic/examples/gothicbook

go 1.26.0

require (
	github.com/metruzanca/gothic v0.3.0
	github.com/metruzanca/gothic/browser v0.0.0
	github.com/metruzanca/gothic/ui v0.1.0
)

require github.com/go-chi/chi/v5 v5.5.0 // indirect

// Local development builds against this checkout. The Railway build drops
// these replaces and fetches the core and UI modules at the deployed commit.
// See railpack.json and README.md.
replace github.com/metruzanca/gothic => ../..

replace github.com/metruzanca/gothic/ui => ../../ui

replace github.com/metruzanca/gothic/browser => ../../browser
