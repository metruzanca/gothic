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
