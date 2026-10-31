package ui

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
)

func virtualItems(n int) []VirtualItem {
	items := make([]VirtualItem, n)
	for i := range items {
		items[i] = VirtualItem{ID: i + 1, Title: fmt.Sprintf("Row %d", i+1), Detail: "detail"}
	}
	return items
}

func TestVirtualListRender(t *testing.T) {
	props := VirtualListProps{Items: virtualItems(100), Height: 72, RowHeight: 36}
	var b bytes.Buffer
	if err := VirtualList("v1", "", props).Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	html := b.String()
	for _, want := range []string{
		`class="ui-virtual-list"`,
		`style="height:72px"`,
		`style="height:0px"`,
		`style="height:3528px"`,
		`Row 1`,
		`Row 2`,
		`"Start":0,"Count":2`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("render missing %q\n\n%s", want, html)
		}
	}
	if got := strings.Count(html, `class="ui-virtual-row"`); got != 2 {
		t.Errorf("rendered %d rows, want 2", got)
	}
}

func TestVirtualListUpdate(t *testing.T) {
	props := VirtualListProps{Items: virtualItems(100), Height: 72, RowHeight: 36}
	m := NewVirtualListModel(props)
	m, _ = m.Update(VirtualScrolled{Top: 360}, props)
	if m.Start != 10 || m.Count != 2 || m.Top != 360 || m.Bottom != 3168 {
		t.Fatalf("scrolled model = %+v", m)
	}

	small := VirtualListProps{Items: virtualItems(3), Height: 72, RowHeight: 36}
	m, _ = m.Update(gothic.PropsChanged{}, small)
	if m.Start != 1 || m.Count != 2 || m.Top != 36 || m.Bottom != 0 {
		t.Fatalf("props changed model = %+v", m)
	}
}
