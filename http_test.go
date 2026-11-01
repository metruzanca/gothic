package gothic_test

import (
	"context"
	"encoding/json"
	"errors"
	. "github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/gothicest"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	var got payload
	if err := DecodeJSON(httptest.NewRecorder(), req, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "x" {
		t.Fatalf("name = %q", got.Name)
	}

	cases := []struct {
		name        string
		contentType string
		body        string
		status      int
	}{
		{"wrong content type", "text/plain", `{}`, http.StatusUnsupportedMediaType},
		{"invalid json", "application/json", `{`, http.StatusBadRequest},
		{"empty body", "application/json", ``, http.StatusBadRequest},
		{"trailing data", "application/json", `{}{}`, http.StatusBadRequest},
		{"json suffix", "application/problem+json", `{}`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			err := DecodeJSON(httptest.NewRecorder(), req, &got)
			if tc.status == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("error = %v, want *Error", err)
			}
			if e.Status != tc.status {
				t.Fatalf("status = %d, want %d", e.Status, tc.status)
			}
		})
	}
}

func TestWriteErrorNegotiation(t *testing.T) {
	failure := Invalid("validation_failed", "check the form", map[string]string{"title": "required"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	WriteError(rec, req, failure)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	var payload struct {
		Error struct {
			Message string            `json:"message"`
			Code    string            `json:"code"`
			Fields  map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != "validation_failed" || payload.Error.Fields["title"] != "required" {
		t.Fatalf("payload = %+v", payload)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	WriteError(rec, req, failure)
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("content type = %q", got)
	}
}

func TestUnmarshalArgsReportsBadRequest(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.RegisterProcedure("Greet", func(ctx context.Context, body []byte) (any, error) {
		var name string
		if err := UnmarshalArgs(body, &name); err != nil {
			return nil, err
		}
		return "hello " + name, nil
	})
	ts := gothicest.NewTestServer(t, s)

	res := ts.PostJSON(RPCPrefix+"Greet", map[string]any{"args": []any{}})
	body := ts.Body(res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %s", res.StatusCode, body)
	}
}
