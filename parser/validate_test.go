package parser_test

import (
	"strings"
	"testing"

	"github.com/metruzanca/gothic/parser"
)

func TestValidateRejectsCollisions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "duplicate procedure",
			src: `package p
rpc Ping() string
rpc Ping() string
templ T() {
	<div></div>
}
`,
			want: "duplicate procedure",
		},
		{
			name: "state and procedure",
			src: `package p
templ T() {
	state Ping = []string{}
	<div></div>
}
rpc Ping() string
`,
			want: "collides with an rpc procedure",
		},
		{
			name: "runtime reserved name",
			src: `package p
templ T() {
	<div></div>
}
rpc signal() string
`,
			want: "reserved by the gothic client runtime",
		},
		{
			name: "javascript keyword",
			src: `package p
templ T() {
	<div></div>
}
rpc class() string
`,
			want: "reserved JavaScript word",
		},
		{
			name: "go keyword",
			src: `package p
templ T() {
	<div></div>
}
rpc return() string
`,
			want: "not a valid Go identifier",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := parser.Parse("test.gothic", []byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			err = parser.Validate(f)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() = %v, want an error containing %q", err, tc.want)
			}
		})
	}
}

func TestValidateAcceptsDemo(t *testing.T) {
	f, err := parser.Parse("demo.gothic", []byte(demo))
	if err != nil {
		t.Fatal(err)
	}
	if err := parser.Validate(f); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}
