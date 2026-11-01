package gothic

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFormValidation(t *testing.T) {
	body := url.Values{"email": {"ada@example.com"}, "remember": {"on"}}.Encode()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, err := ParseForm(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got := form.Required("email", "required"); got != "ada@example.com" {
		t.Fatalf("email = %q", got)
	}
	if !form.Bool("remember") {
		t.Fatal("checkbox not read")
	}
	if err := form.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	form.Required("password", "required")
	var validation *Error
	if !errors.As(form.Err(), &validation) {
		t.Fatalf("error = %v, want *Error", form.Err())
	}
	if validation.Fields["password"] != "required" {
		t.Fatalf("fields = %#v", validation.Fields)
	}
}

func TestParseFormRejectsNonFormBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	_, err := ParseForm(httptest.NewRecorder(), req)
	var e *Error
	if !errors.As(err, &e) || e.Status != http.StatusUnsupportedMediaType {
		t.Fatalf("error = %v", err)
	}
}
