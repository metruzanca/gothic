module github.com/metruzanca/gothic/examples/gothicbook

go 1.26.0

require (
	github.com/metruzanca/gothic v0.2.0
	github.com/metruzanca/gothic/ui v0.0.0
)

require github.com/go-chi/chi/v5 v5.5.0 // indirect

replace github.com/metruzanca/gothic => ../..

replace github.com/metruzanca/gothic/ui => ../../ui
