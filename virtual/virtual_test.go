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

func TestVariableWindow(t *testing.T) {
	heights := map[int]int{0: 50, 1: 30}
	cases := []struct {
		name                          string
		scrollTop, estimate, viewport int
		total                         int
		want                          Window
	}{
		{"top", 0, 40, 100, 4, Window{Start: 0, Count: 2, Top: 0, Bottom: 80}},
		{"middle", 60, 40, 100, 4, Window{Start: 1, Count: 2, Top: 50, Bottom: 40}},
		{"end", 999, 40, 100, 4, Window{Start: 3, Count: 1, Top: 120, Bottom: 0}},
		{"empty", 0, 40, 100, 0, Window{}},
		{"default estimate", 0, 0, 100, 3, Window{Start: 0, Count: 2, Top: 0, Bottom: 40}},
	}
	for _, tc := range cases {
		if got := VariableWindow(heights, tc.estimate, tc.scrollTop, tc.viewport, tc.total); got != tc.want {
			t.Errorf("%s: VariableWindow(%v, %d, %d, %d, %d) = %+v, want %+v", tc.name, heights, tc.estimate, tc.scrollTop, tc.viewport, tc.total, got, tc.want)
		}
	}
}

func TestVariableWindowMeasuredRows(t *testing.T) {
	heights := map[int]int{2: 120}
	got := VariableWindow(heights, 40, 80, 100, 5)
	if got.Start != 2 || got.Count != 1 || got.Top != 80 || got.Bottom != 80 {
		t.Fatalf("window = %+v", got)
	}
}

func TestWindowForRows(t *testing.T) {
	fixed := WindowForRows(nil, 36, 40, 3600, 432, 5000)
	if fixed != (Window{Start: 100, Count: 12, Top: 3600, Bottom: 175968}) {
		t.Fatalf("fixed window = %+v", fixed)
	}
	variable := WindowForRows(map[int]int{0: 50}, 0, 40, 0, 100, 4)
	if variable != (Window{Start: 0, Count: 2, Top: 0, Bottom: 80}) {
		t.Fatalf("variable window = %+v", variable)
	}
}
