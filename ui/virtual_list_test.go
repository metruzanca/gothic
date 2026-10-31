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
