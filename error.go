package gothic

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is an HTTP-aware error. Procedures and handlers return it to control
// the status code, the machine-readable code, and per-field messages that
// clients receive.
//
// Plain errors are treated as internal server errors. Their message is kept
// for compatibility, so domain errors returned by procedures still reach the
// client. Wrap sensitive errors in an Error with a safe message instead.
type Error struct {
	// Status is the HTTP status code. It defaults to 500.
	Status int
	// Code is a stable machine-readable identifier, such as
	// "invalid_argument". Clients can branch on it.
	Code string
	// Message is sent to the client.
	Message string
	// Fields maps a field name to a validation message.
	Fields map[string]string
	// Err is the wrapped cause. It is logged and never sent to the client.
	Err error
}

// Error implements error.
func (e *Error) Error() string {
	switch {
	case e == nil:
		return "<nil>"
	case e.Message != "":
		return e.Message
	case e.Err != nil:
		return e.Err.Error()
	case e.Code != "":
		return e.Code
	default:
		return http.StatusText(e.status())
	}
}

// Unwrap returns the wrapped cause.
func (e *Error) Unwrap() error { return e.Err }

func (e *Error) status() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	if e.Status != 0 {
		return e.Status
	}
	return http.StatusInternalServerError
}

// Errorf builds an Error with a formatted message.
func Errorf(status int, code, format string, args ...any) *Error {
	return &Error{Status: status, Code: code, Message: fmt.Sprintf(format, args...)}
}

// BadRequest reports a malformed or invalid request.
func BadRequest(message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "invalid_argument", Message: message}
}

// Invalid reports a validation failure with per-field messages.
func Invalid(code, message string, fields map[string]string) *Error {
	if code == "" {
		code = "invalid_argument"
	}
	return &Error{Status: http.StatusBadRequest, Code: code, Message: message, Fields: fields}
}

// Unauthorized reports a missing or invalid credential.
func Unauthorized(message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "unauthenticated", Message: message}
}

// Forbidden reports an authenticated request that is not allowed.
func Forbidden(message string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "permission_denied", Message: message}
}

// NotFound reports a missing resource.
func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

// Conflict reports a request that conflicts with the current state.
func Conflict(message string) *Error {
	return &Error{Status: http.StatusConflict, Code: "already_exists", Message: message}
}

// Internal reports a server failure with a safe client message.
func Internal(message string, err error) *Error {
	if message == "" {
		message = "internal server error"
	}
	return &Error{Status: http.StatusInternalServerError, Code: "internal", Message: message, Err: err}
}

// asError converts any error to an Error.
func asError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{Status: http.StatusInternalServerError, Message: err.Error(), Err: err}
}
