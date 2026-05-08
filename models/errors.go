package models

import (
	"errors"
	"fmt"
	"net/http"
)

// HTTPError is the small error-with-status-code helper that controllers and
// the global error handler look for. It plays the same role as `guru.Error`
// in deskapi: services return an HTTPError whenever a failure should map to
// a specific HTTP status, and the controller layer doesn't have to translate.
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}

// NewHTTPError is the canonical constructor.
func NewHTTPError(status int, msg string) *HTTPError {
	return &HTTPError{Status: status, Message: msg}
}

// AsHTTPError extracts an *HTTPError if there is one in the chain.
func AsHTTPError(err error) (*HTTPError, bool) {
	var he *HTTPError
	if errors.As(err, &he) {
		return he, true
	}
	return nil, false
}

// ErrNotFound is the canonical 404. Use NewHTTPError directly for richer
// messages.
var ErrNotFound = NewHTTPError(http.StatusNotFound, "not found")
