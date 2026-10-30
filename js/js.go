// Package js exposes a small, curated set of browser APIs to gothic apps.
//
// Every function is client-only. Call them from Update or from commands. The
// Go stubs exist so the prelude compiles; the generator lowers each call to
// the equivalent browser API in the generated JavaScript.
package js

const clientOnly = "gothic: js functions can only run in the browser"

// Alert shows a browser alert dialog.
func Alert(message string) {
	panic(clientOnly)
}

// Log writes values to the browser console.
func Log(values ...any) {
	panic(clientOnly)
}

// SetTitle sets the document title.
func SetTitle(title string) {
	panic(clientOnly)
}

// GetItem reads a localStorage value. It returns an empty string when the key
// is missing.
func GetItem(key string) string {
	panic(clientOnly)
}

// SetItem writes a localStorage value.
func SetItem(key, value string) {
	panic(clientOnly)
}

// RemoveItem removes a localStorage value.
func RemoveItem(key string) {
	panic(clientOnly)
}

// Copy writes text to the clipboard.
func Copy(text string) {
	panic(clientOnly)
}

// Focus focuses the first element matching the CSS selector.
func Focus(selector string) {
	panic(clientOnly)
}

// Reload reloads the current page.
func Reload() {
	panic(clientOnly)
}

// Navigate pushes a new path onto browser history and dispatches
// LocationChanged. The app must define a LocationChanged message.
func Navigate(path string) {
	panic(clientOnly)
}

// Replace replaces the current history entry and dispatches LocationChanged.
// The app must define a LocationChanged message.
func Replace(path string) {
	panic(clientOnly)
}

// Back moves one entry back in browser history. The resulting popstate
// dispatches LocationChanged.
func Back() {
	panic(clientOnly)
}
