// Package virtual contains headless windowing helpers for long lists.
//
// WindowFor maps a scroll offset to the slice of rows that should exist in the
// DOM. The same rules run on the server and in the generated client code.
package virtual

// Window is the visible slice of a list.
type Window struct {
	Start  int
	Count  int
	Top    int
	Bottom int
}

// WindowFor returns the visible window for a list of total rows.
//
// scrollTop is the pixel offset of the scroll container, rowHeight is the
// fixed height of one row, and viewportHeight is the height of the visible
// area. Top and Bottom are the spacer heights before and after the window.
func WindowFor(scrollTop, rowHeight, viewportHeight, total int) Window {
	if rowHeight <= 0 || total <= 0 {
		return Window{}
	}
	count := viewportHeight / rowHeight
	if count < 1 {
		count = 1
	}
	if count > total {
		count = total
	}
	start := scrollTop / rowHeight
	if start < 0 {
		start = 0
	}
	maxStart := total - count
	if start > maxStart {
		start = maxStart
	}
	return Window{
		Start:  start,
		Count:  count,
		Top:    start * rowHeight,
		Bottom: (total - start - count) * rowHeight,
	}
}
