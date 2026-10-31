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

// VariableWindow returns the visible window when rows can have different
// heights. heights maps a row index to its measured height; missing or zero
// entries use estimate.
func VariableWindow(heights map[int]int, estimate, scrollTop, viewportHeight, total int) Window {
	if total <= 0 {
		return Window{}
	}
	if estimate <= 0 {
		estimate = 40
	}
	if viewportHeight <= 0 {
		viewportHeight = 1
	}
	if scrollTop < 0 {
		scrollTop = 0
	}
	heightAt := func(i int) int {
		if h, ok := heights[i]; ok && h > 0 {
			return h
		}
		return estimate
	}
	top := 0
	start := 0
	for i := 0; i < total; i++ {
		h := heightAt(i)
		if top+h > scrollTop {
			start = i
			break
		}
		top += h
		start = i + 1
	}
	if start >= total {
		start = total - 1
		top = 0
		for i := 0; i < start; i++ {
			top += heightAt(i)
		}
	}
	count := 0
	used := 0
	for i := start; i < total; i++ {
		h := heightAt(i)
		if used > 0 && used+h > viewportHeight {
			break
		}
		used += h
		count++
		if used >= viewportHeight {
			break
		}
	}
	if count < 1 {
		count = 1
	}
	bottom := 0
	for i := start + count; i < total; i++ {
		bottom += heightAt(i)
	}
	return Window{Start: start, Count: count, Top: top, Bottom: bottom}
}

// WindowForRows returns the visible window for a list with fixed or variable
// row heights. A positive rowHeight selects fixed mode; zero selects variable
// mode using estimate.
func WindowForRows(heights map[int]int, rowHeight, estimate, scrollTop, viewportHeight, total int) Window {
	if rowHeight > 0 {
		return WindowFor(scrollTop, rowHeight, viewportHeight, total)
	}
	return VariableWindow(heights, estimate, scrollTop, viewportHeight, total)
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
