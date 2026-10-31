package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
)

func TestNestedApp(t *testing.T) {
	s := gothic.NewServer()
	s.MountApp("/", gothic.NewApp(NewModel()))
	ts := httptest.NewServer(s)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	for _, want := range []string{
		`data-gothic-root="app"`,
		`"ShowDashboard":true`,
		`data-gothic-component="Dashboard"`,
		`"ShowBranch":false,"Items":["alpha","beta"]`,
		`data-gothic-component="Counter"`,
		`>static<`,
		`>alpha<`,
		`>beta<`,
		`"Count":0`,
		`Nested dashboard`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/app_components_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	js, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, want := range []string{
		`const c0 = mountCounter(c0Node, {Label: "static", OnPick: {tag: "PickedChild", Label: "static"}}, emit, (msg) => dispatch({tag: "ChildMessage"`,
		`createCounter("c1", {Label: "branch", OnPick: {tag: "PickedChild", Label: "branch"}})`,
		`listLocal(root, "0",`,
		`mountCounter(node, {Label: item, OnPick: {tag: "PickedChild", Label: item}}, emit, (msg) => dispatch({tag: "ChildMessage"`,
	} {
		if !strings.Contains(string(js), want) {
			t.Errorf("component JS missing %q\n\n%s", want, js)
		}
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/app_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	appJS, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, want := range []string{
		`from "/_gothic/modules/app_components_gothic.js"`,
		`mountDashboard`,
		`createDashboard`,
	} {
		if !strings.Contains(string(appJS), want) {
			t.Errorf("app JS missing %q\n\n%s", want, appJS)
		}
	}
}
