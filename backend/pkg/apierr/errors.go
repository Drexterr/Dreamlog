package apierr

import (
	"errors"
	"net/http"
)

// APIError is a structured error returned to clients.
type APIError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
	// Cause is the original internal error, if any. Never serialized to the
	// client (json:"-") - it exists purely so middleware.ErrorHandler can log
	// and report the real failure instead of just the client-facing message.
	Cause error `json:"-"`
}

func (e *APIError) Error() string { return e.Message }

// Unwrap exposes Cause to errors.Is/errors.As chains.
func (e *APIError) Unwrap() error { return e.Cause }

// Sentinel constructors.
func New(code int, message string, detail ...string) *APIError {
	e := &APIError{Code: code, Message: message}
	if len(detail) > 0 {
		e.Detail = detail[0]
	}
	return e
}

func BadRequest(msg string, detail ...string) *APIError {
	return New(http.StatusBadRequest, msg, detail...)
}

func Unauthorized(msg string) *APIError {
	return New(http.StatusUnauthorized, msg)
}

func Forbidden(msg string) *APIError {
	return New(http.StatusForbidden, msg)
}

func NotFound(resource string) *APIError {
	return New(http.StatusNotFound, resource+" not found")
}

func Conflict(msg string) *APIError {
	return New(http.StatusConflict, msg)
}

func Internal(msg string) *APIError {
	return New(http.StatusInternalServerError, msg)
}

// InternalErr is Internal, but carries the original error so
// middleware.ErrorHandler can log and report the real cause instead of just
// the generic client-facing message. Use this whenever a real error is being
// converted to a 500 - the bare Internal(msg) form leaves that error
// untraceable in logs and Sentry.
func InternalErr(msg string, cause error) *APIError {
	e := Internal(msg)
	e.Cause = cause
	return e
}

// As unwraps an *APIError from any error chain.
func As(err error) (*APIError, bool) {
	var e *APIError
	ok := errors.As(err, &e)
	return e, ok
}
