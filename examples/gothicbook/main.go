//go:generate go run github.com/metruzanca/gothic/cmd/gothic generate .

package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/metruzanca/gothic"
)

//go:embed style.css
var pageCSS []byte

//go:embed assets/gothic-mark.svg
var pageMark []byte

//go:embed assets/gothic-wordmark.svg
var pageWordmark []byte

//go:embed assets/og.png
var pageOG []byte

// Asset URLs carry a content hash so a deploy is never served a stale
// stylesheet or social card from the Railway edge cache.
var (
	styleHref    = assetHref("/style.css", pageCSS)
	ogImageHref  = assetHref("/og.png", pageOG)
	wordmarkHref = assetHref("/gothic-wordmark.svg", pageWordmark)
)

func assetHref(path string, data []byte) string {
	sum := sha256.Sum256(data)
	return path + "?v=" + hex.EncodeToString(sum[:6])
}

// themeScript flips the color theme for the current page load. The choice is
// deliberately not stored, so a reload returns to the system setting.
const themeScript = `<script>
(() => {
	const root = document.documentElement;
	const media = window.matchMedia('(prefers-color-scheme: dark)');
	document.addEventListener('click', (event) => {
		if (!(event.target instanceof Element)) return;
		const button = event.target.closest('[data-theme-toggle]');
		if (!button) return;
		const current = root.dataset.theme || (media.matches ? 'dark' : 'light');
		root.dataset.theme = current === 'dark' ? 'light' : 'dark';
	});
})();
</script>`

type counterStore struct {
	count atomic.Int64
}

func (c *counterStore) increment() int {
	return int(c.count.Add(1))
}

type procedures struct {
	counter *counterStore
}

func (p procedures) Increment(ctx context.Context) (int, error) {
	return p.counter.increment(), nil
}

func main() {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	log.Fatal(s.ListenAndServe(addr))
}

func mountApp(s *gothic.Server, counter *counterStore) {
	RegisterProcedures(s, procedures{counter: counter})
	s.Asset("/style.css", pageCSS)
	s.Asset("/gothic-mark.svg", pageMark)
	s.Asset("/gothic-wordmark.svg", pageWordmark)
	s.Asset("/og.png", pageOG)
	s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
		return NewModelAt(r.URL.Path, int(counter.count.Load()))
	}),
		gothic.WithDocument(gothic.Document{
			Title:       "Gothicbook",
			SiteName:    "Gothic",
			Description: "A tiny design system demo for the installable gothic UI kit.",
			Image:       ogImageHref,
			Icons:       []gothic.Icon{{Href: "/gothic-mark.svg", Type: "image/svg+xml"}},
			Styles:      []string{styleHref},
			ExtraHead:   themeScript,
		}),
	)
}
