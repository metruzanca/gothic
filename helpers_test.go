package gothic_test

import (
	"io"
	"net/http"
	"testing"
)

func mustRequest(t testing.TB, method, target string, headers map[string]string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req
}
