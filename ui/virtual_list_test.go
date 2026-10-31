package ui

import (
	"testing"

	"github.com/metruzanca/gothic"
)

func virtualItems(n int) []any {
	items := make([]any, n)
	for i := range items {
		items[i] = i + 1
	}
	return items
}

func TestVirtualListUpdate(t *testing.T) {
	props := VirtualListProps{Items: virtualItems(100), Height: 72, RowHeight: 36}
	m := NewVirtualListModel(props)
	m, _ = m.Update(VirtualScrolled{Top: 360}, props)
	if m.Start != 10 || m.Count != 2 || m.Top != 360 || m.Bottom != 3168 {
		t.Fatalf("scrolled model = %+v", m)
	}
	if m.TopStyle != "height:360px" || m.BottomStyle != "height:3168px" {
		t.Fatalf("scrolled styles = %+v", m)
	}

	small := VirtualListProps{Items: virtualItems(3), Height: 72, RowHeight: 36}
	m, _ = m.Update(gothic.PropsChanged{}, small)
	if m.Start != 1 || m.Count != 2 || m.Top != 36 || m.Bottom != 0 {
		t.Fatalf("props changed model = %+v", m)
	}
}

func TestVirtualListResponsive(t *testing.T) {
	props := VirtualListProps{Items: virtualItems(200), Height: 216, RowHeight: 36}
	m := NewVirtualListModel(props)
	m, _ = m.Update(VirtualResized{Height: 300}, props)
	if m.Height != 300 || m.Count != 8 {
		t.Fatalf("resized model = %+v", m)
	}
	if m.HeightStyle != "height:216px" {
		t.Fatalf("fixed height style changed: %q", m.HeightStyle)
	}

	auto := VirtualListProps{Items: virtualItems(200), Height: 0, RowHeight: 36}
	a := NewVirtualListModel(auto)
	if a.HeightStyle != "" {
		t.Fatalf("auto height style = %q", a.HeightStyle)
	}
	a, _ = a.Update(VirtualResized{Height: 180}, auto)
	if a.Height != 180 || a.Count != 5 {
		t.Fatalf("auto resized model = %+v", a)
	}
}

func TestVirtualListVariableRows(t *testing.T) {
	estimate := 50
	props := VirtualListProps{Items: virtualItems(200), Height: 216, RowHeight: 0, Estimate: &estimate}
	m := NewVirtualListModel(props)
	if m.RowHeight != 0 || m.RowHeightStyle != "" || m.Estimate != 50 {
		t.Fatalf("variable model = %+v", m)
	}
	if m.Count != 4 {
		t.Fatalf("count = %d, want 4", m.Count)
	}
	m, _ = m.Update(VirtualRowMeasured{Index: 0, Height: 80}, props)
	if m.Heights[0] != 80 {
		t.Fatalf("heights = %+v", m.Heights)
	}
	m, _ = m.Update(VirtualResized{Height: 320}, props)
	if m.Height != 320 || m.Count < 5 {
		t.Fatalf("resized variable model = %+v", m)
	}
}
