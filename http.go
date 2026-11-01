package gothic

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

// DefaultMaxBodyBytes is the body limit for procedures and DecodeJSON.
const DefaultMaxBodyBytes int64 = 1 << 20

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// DecodeJSON decodes a JSON request body with the default body limit. It
// requires an application/json content type and returns an *Error on failure.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	return DecodeJSONLimit(w, r, v, DefaultMaxBodyBytes)
}

// DecodeJSONLimit is DecodeJSON with an explicit body limit.
func DecodeJSONLimit(w http.ResponseWriter, r *http.Request, v any, limit int64) error {
	if !IsJSONContentType(r.Header.Get("Content-Type")) {
		return Errorf(http.StatusUnsupportedMediaType, "unsupported_media_type", "expected application/json")
	}
	if limit <= 0 {
		limit = DefaultMaxBodyBytes
	}
	body := http.MaxBytesReader(w, r.Body, limit)
	dec := json.NewDecoder(body)
	if err := dec.Decode(v); err != nil {
		if maxErr := (*http.MaxBytesError)(nil); errors.As(err, &maxErr) {
			return Errorf(http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds %d bytes", limit)
		}
		if errors.Is(err, io.EOF) {
			return BadRequest("request body is empty")
		}
		return BadRequest("invalid JSON body: " + err.Error())
	}
	if dec.More() {
		return BadRequest("invalid JSON body: unexpected trailing data")
	}
	return nil
}

// IsJSONContentType reports whether value is application/json or a structured
// JSON suffix such as application/problem+json.
func IsJSONContentType(value string) bool {
	if value == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

// WriteError writes err as an HTTP response. JSON clients get the error
// envelope; other clients get plain text.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	e := asError(err)
	if wantsJSON(r) {
		writeErrorJSON(w, e)
		return
	}
	http.Error(w, e.Error(), e.status())
}

// writeErrorJSON writes the error envelope. Procedures always use it.
func writeErrorJSON(w http.ResponseWriter, e *Error) {
	status := e.status()
	detail := map[string]any{"message": e.Error()}
	if e.Code != "" {
		detail["code"] = e.Code
	}
	if len(e.Fields) > 0 {
		detail["fields"] = e.Fields
	}
	_ = JSON(w, status, map[string]any{"error": detail})
}

func wantsJSON(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.HasPrefix(r.URL.Path, frameworkPrefix) {
		return true
	}
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}
	return strings.Contains(accept, "application/json") || strings.Contains(accept, "+json")
}
