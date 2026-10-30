package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
)

func TestRouterDeepLinks(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s)
	ts := httptest.NewServer(s)
	defer ts.Close()

	for _, tc := range []struct {
		path string
		want string
	}{
		{"/", "<h1>Home</h1>"},
		{"/about", "<h1>About</h1>"},
		{"/settings", "<h1>Settings</h1>"},
		{"/missing", "<h1>Not found</h1>"},
	} {
		res, err := http.Get(ts.URL + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d", tc.path, res.StatusCode)
		}
		if !strings.Contains(string(body), tc.want) {
			t.Errorf("GET %s missing %q\n\n%s", tc.path, tc.want, body)
		}
		if !strings.Contains(string(body), "Current path:") {
			t.Errorf("GET %s missing current path\n\n%s", tc.path, body)
		}
	}

	res, err := http.Get(ts.URL + "/missing")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	stripped := regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(string(body), "")
	if !strings.Contains(stripped, "No route for /missing.") {
		t.Errorf("missing page did not render the path\n\n%s", stripped)
	}
}

func TestRouterModule(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s)
	ts := httptest.NewServer(s)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/_gothic/modules/app_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	js := string(body)
	for _, want := range []string{
		`routeClick`,
		`window.addEventListener("popstate"`,
		`dispatch(locationMessage());`,
		`navigate("/settings")`,
		`localStorage.getItem("lastPath")`,
		`localStorage.setItem("lastPath", msg.Path)`,
		`document.title`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("app JS missing %q\n\n%s", want, js)
		}
	}
}
