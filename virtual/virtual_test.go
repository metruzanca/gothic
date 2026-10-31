package virtual

import "testing"

func TestWindowFor(t *testing.T) {
	cases := []struct {
		name                           string
		scrollTop, rowHeight, viewport int
		total                          int
		want                           Window
	}{
		{"top", 0, 36, 432, 5000, Window{Start: 0, Count: 12, Top: 0, Bottom: 179568}},
		{"snapped", 3600, 36, 432, 5000, Window{Start: 100, Count: 12, Top: 3600, Bottom: 175968}},
		{"between rows", 3610, 36, 432, 5000, Window{Start: 100, Count: 12, Top: 3600, Bottom: 175968}},
		{"negative", -50, 36, 432, 5000, Window{Start: 0, Count: 12, Top: 0, Bottom: 179568}},
		{"end", 999999, 36, 432, 5000, Window{Start: 4988, Count: 12, Top: 179568, Bottom: 0}},
		{"short list", 0, 36, 432, 5, Window{Start: 0, Count: 5, Top: 0, Bottom: 0}},
		{"empty", 0, 36, 432, 0, Window{}},
		{"bad row height", 0, 0, 432, 10, Window{}},
		{"tiny viewport", 0, 36, 10, 10, Window{Start: 0, Count: 1, Top: 0, Bottom: 324}},
	}
	for _, tc := range cases {
		if got := WindowFor(tc.scrollTop, tc.rowHeight, tc.viewport, tc.total); got != tc.want {
			t.Errorf("%s: WindowFor(%d, %d, %d, %d) = %+v, want %+v", tc.name, tc.scrollTop, tc.rowHeight, tc.viewport, tc.total, got, tc.want)
		}
	}
}
