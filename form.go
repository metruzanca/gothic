package gothic

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// Form wraps parsed form values and collects field errors.
type Form struct {
	values url.Values
	fields map[string]string
}

// ParseForm parses an HTML form with the default body limit and returns a Form.
func ParseForm(w http.ResponseWriter, r *http.Request) (*Form, error) {
	if !isFormContentType(r.Header.Get("Content-Type")) {
		return nil, Errorf(http.StatusUnsupportedMediaType, "unsupported_media_type", "expected a form body")
	}
	r.Body = http.MaxBytesReader(w, r.Body, DefaultMaxBodyBytes)
	if err := r.ParseForm(); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, Errorf(http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds %d bytes", DefaultMaxBodyBytes)
		}
		return nil, BadRequest("invalid form data")
	}
	return &Form{values: r.PostForm}, nil
}

// String returns a trimmed value.
func (f *Form) String(name string) string {
	return strings.TrimSpace(f.values.Get(name))
}

// Bool reports whether a checkbox value is set.
func (f *Form) Bool(name string) bool {
	switch strings.ToLower(f.values.Get(name)) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// Required returns a trimmed value and records a field error when it is empty.
func (f *Form) Required(name, message string) string {
	value := f.String(name)
	if value == "" {
		f.AddError(name, message)
	}
	return value
}

// AddError records a field error.
func (f *Form) AddError(name, message string) {
	if f.fields == nil {
		f.fields = map[string]string{}
	}
	if _, ok := f.fields[name]; ok {
		return
	}
	f.fields[name] = message
}

// FieldError returns the error for a field.
func (f *Form) FieldError(name string) string {
	return f.fields[name]
}

// Err returns nil when the form is valid, or a validation error with fields.
func (f *Form) Err() error {
	if len(f.fields) == 0 {
		return nil
	}
	return Invalid("validation_failed", "check the form", f.fields)
}

// Values returns the parsed values.
func (f *Form) Values() url.Values { return f.values }

func isFormContentType(value string) bool {
	return strings.HasPrefix(strings.ToLower(value), "application/x-www-form-urlencoded")
}
