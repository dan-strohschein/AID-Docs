// Package errors demonstrates error type extraction.
package errors

import "fmt"

// StatusCode represents an HTTP status code.
type StatusCode int

const (
	// StatusOK indicates success.
	StatusOK StatusCode = iota
	// StatusNotFound indicates the resource was not found.
	StatusNotFound
	// StatusForbidden indicates access is denied.
	StatusForbidden
	// StatusError indicates a server error.
	StatusError
)

// NotFoundError is returned when a resource doesn't exist.
type NotFoundError struct {
	Resource string
	ID       string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
}

// ValidationError is returned when input is invalid.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on %s: %s", e.Field, e.Message)
}

// Wrap wraps an error with additional context.
func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}
