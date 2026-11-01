package gothic

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDevToolsWorkspaceDisabled(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.MountApp("/", NewApp(baseModel{}))
	ts := NewTestServer(t, s)

	res := ts.Get(DevToolsWorkspacePath)
	body := ts.Body(res)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body %s", res.StatusCode, body)
	}
	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("content type = %q", res.Header.Get("Content-Type"))
	}
	if strings.Contains(body, "<html") {
		t.Fatalf("probe reached the app: %s", body)
	}
}

func TestDevToolsWorkspaceEnabled(t *testing.T) {
	dir := t.TempDir()
	s := NewServer(WithLogger(testLogger()), WithDevMode(dir))
	s.MountApp("/", NewApp(baseModel{}))
	ts := NewTestServer(t, s)

	res := ts.Get(DevToolsWorkspacePath)
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; body %s", res.StatusCode, body)
	}
	var payload struct {
		Workspace struct {
			Root string `json:"root"`
			UUID string `json:"uuid"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Workspace.Root != dir {
		t.Fatalf("root = %q, want %q", payload.Workspace.Root, dir)
	}
	if payload.Workspace.UUID == "" {
		t.Fatal("missing uuid")
	}
	if payload.Workspace.UUID != workspaceUUID(dir) {
		t.Fatalf("uuid = %q, want stable uuid", payload.Workspace.UUID)
	}

	res = ts.Do(mustRequest(t, http.MethodHead, ts.URL+DevToolsWorkspacePath, nil, nil))
	body = ts.Body(res)
	if res.StatusCode != http.StatusOK || body != "" {
		t.Fatalf("HEAD = %d %q", res.StatusCode, body)
	}
}

func TestDevToolsUUIDIsStableAndDirectorySpecific(t *testing.T) {
	first := workspaceUUID("/projects/one")
	if first != workspaceUUID("/projects/one") {
		t.Fatal("uuid is not stable")
	}
	if first == workspaceUUID("/projects/two") {
		t.Fatal("different directories share a uuid")
	}
	if len(first) != 36 {
		t.Fatalf("uuid = %q, want 36 characters", first)
	}
}

func TestDevToolsBypassesMiddleware(t *testing.T) {
	var hits int
	s := NewServer(WithLogger(testLogger()), WithDevMode(t.TempDir()))
	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			WriteError(w, r, Unauthorized("sign in"))
		})
	})
	ts := NewTestServer(t, s)

	res := ts.Get(DevToolsWorkspacePath)
	ts.Body(res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if hits != 0 {
		t.Fatalf("middleware ran %d times", hits)
	}
}
