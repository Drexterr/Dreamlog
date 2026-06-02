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
}

func (e *APIError) Error() string { return e.Message }

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

// As unwraps an *APIError from any error chain.
func As(err error) (*APIError, bool) {
	var e *APIError
	ok := errors.As(err, &e)
	return e, ok
}
