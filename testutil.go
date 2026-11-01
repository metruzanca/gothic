package gothic

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

// TestServer wraps a Server behind httptest.Server and closes it with the test.
type TestServer struct {
	*httptest.Server
	Gothic *Server

	t testing.TB
}

// NewTestServer starts s on a random port and closes it when the test ends.
func NewTestServer(t testing.TB, s *Server) *TestServer {
	t.Helper()
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)
	client := ts.Client()
	client.Jar, _ = cookiejar.New(nil)
	return &TestServer{Server: ts, Gothic: s, t: t}
}

// Do sends req and fails the test on a transport error.
func (ts *TestServer) Do(req *http.Request) *http.Response {
	ts.t.Helper()
	res, err := ts.Client().Do(req)
	if err != nil {
		ts.t.Fatalf("request failed: %v", err)
	}
	return res
}

// Get sends a GET request and returns the response.
func (ts *TestServer) Get(path string) *http.Response {
	ts.t.Helper()
	return ts.Do(mustRequest(ts.t, http.MethodGet, ts.URL+path, nil, nil))
}

// PostJSON sends a JSON body and returns the response.
func (ts *TestServer) PostJSON(path string, body any) *http.Response {
	ts.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			ts.t.Fatalf("encode body: %v", err)
		}
	}
	headers := map[string]string{"Content-Type": "application/json"}
	return ts.Do(mustRequest(ts.t, http.MethodPost, ts.URL+path, headers, &buf))
}

// Body reads and closes the response body, failing the test on a read error.
func (ts *TestServer) Body(res *http.Response) string {
	ts.t.Helper()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		ts.t.Fatalf("read body: %v", err)
	}
	return string(data)
}

// Call invokes a procedure and decodes its result. A server error is returned
// as an *Error carrying the status, code, and fields.
func (ts *TestServer) Call(name string, args ...any) (json.RawMessage, error) {
	ts.t.Helper()
	if args == nil {
		args = []any{}
	}
	res := ts.PostJSON(RPCPrefix+name, map[string]any{"args": args})
	defer res.Body.Close()

	var payload struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string            `json:"message"`
			Code    string            `json:"code"`
			Fields  map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Error != nil {
		return nil, &Error{
			Status:  res.StatusCode,
			Code:    payload.Error.Code,
			Message: payload.Error.Message,
			Fields:  payload.Error.Fields,
		}
	}
	if res.StatusCode >= 400 {
		return nil, Errorf(res.StatusCode, "", "procedure %s failed with status %d", name, res.StatusCode)
	}
	return payload.Result, nil
}

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
