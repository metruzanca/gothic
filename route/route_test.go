package route

import "testing"

func TestParams(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    map[string]string
		matched bool
	}{
		{"/", "/", map[string]string{}, true},
		{"/about", "/about", map[string]string{}, true},
		{"/about", "/contact", nil, false},
		{"/todos/:id", "/todos/42", map[string]string{"id": "42"}, true},
		{"/todos/:id", "/todos/", nil, false},
		{"/todos/:id", "/todos/1/2", nil, false},
		{"/users/:name/posts/:post", "/users/ada/posts/7", map[string]string{"name": "ada", "post": "7"}, true},
		{"/users/:name/posts/:post", "/users/ada/posts", nil, false},
		{"/todos/:id", "/todos", nil, false},
	}
	for _, c := range cases {
		got := Params(c.pattern, c.path)
		if c.matched != (got != nil) {
			t.Errorf("Params(%q, %q) matched = %v, want %v", c.pattern, c.path, got != nil, c.matched)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("Params(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
			continue
		}
		for k, v := range c.want {
			if got[k] != v {
				t.Errorf("Params(%q, %q)[%q] = %q, want %q", c.pattern, c.path, k, got[k], v)
			}
		}
	}
}

func TestMatch(t *testing.T) {
	if !Match("/todos/:id", "/todos/42") {
		t.Fatal("expected match")
	}
	if Match("/todos/:id", "/about") {
		t.Fatal("expected no match")
	}
}

func TestResolve(t *testing.T) {
	routes := []Route{
		{Pattern: "/", Page: "home"},
		{Pattern: "/about", Page: "about"},
		{Pattern: "/hello/:name", Page: "hello"},
	}
	for _, tc := range []struct {
		path    string
		matched bool
		page    string
		param   string
	}{
		{"/", true, "home", ""},
		{"/about", true, "about", ""},
		{"/hello/ada", true, "hello", "ada"},
		{"/missing", false, "", ""},
	} {
		got := Resolve(routes, tc.path)
		if got.Matched != tc.matched || got.Page != tc.page {
			t.Errorf("Resolve(%q) = %+v", tc.path, got)
		}
		if tc.param != "" && got.Params["name"] != tc.param {
			t.Errorf("Resolve(%q) name = %q, want %q", tc.path, got.Params["name"], tc.param)
		}
	}
}

func TestJoin(t *testing.T) {
	base := []Route{
		{Pattern: "/", Page: "home"},
		{Pattern: "/users/:id", Page: "user"},
	}
	got := Join("/admin", base)
	if got[0].Pattern != "/admin/" || got[1].Pattern != "/admin/users/:id" {
		t.Fatalf("Join = %+v", got)
	}
	if base[0].Pattern != "/" {
		t.Fatalf("Join mutated input: %+v", base)
	}
}
