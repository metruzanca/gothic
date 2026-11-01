package dom

import "testing"

func TestPx(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want string
	}{
		{0, "0px"},
		{36, "36px"},
		{-4, "-4px"},
	} {
		if got := Px(tc.in); got != tc.want {
			t.Errorf("Px(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStringHelpers(t *testing.T) {
	value := "ready"
	if got := Str(&value); got != "ready" {
		t.Fatalf("Str = %q", got)
	}
	if got := Str(nil); got != "" {
		t.Fatalf("Str(nil) = %q", got)
	}
	if got := StrOr(nil, "fallback"); got != "fallback" {
		t.Fatalf("StrOr = %q", got)
	}
	yes := true
	no := false
	if got := True(&yes); got != "true" {
		t.Fatalf("True(true) = %q", got)
	}
	if got := True(&no); got != "false" {
		t.Fatalf("True(false) = %q", got)
	}
	if got := True(nil); got != "false" {
		t.Fatalf("True(nil) = %q", got)
	}
	if got := If(true, "yes", "no"); got != "yes" {
		t.Fatalf("If = %q", got)
	}
	if got := ID("tab", 3); got != "tab-3" {
		t.Fatalf("ID = %q", got)
	}
}
