// Package model defines domain types and error values shared across layers.
package model

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors returned by the repository layer.
// Services and handlers use errors.Is() to check for these.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrDuplicate is returned when a unique constraint would be violated.
	ErrDuplicate = errors.New("duplicate")
	// ErrConflict is returned when an operation conflicts with current state
	// (e.g. selling more shares than owned).
	ErrConflict = errors.New("conflict")
	// ErrForbidden is returned when the caller lacks permission for the resource.
	ErrForbidden = errors.New("forbidden")
	// ErrValidation is returned when input fails business rule validation.
	ErrValidation = errors.New("validation")
)

// AppError wraps a sentinel error with a human-readable message and HTTP status.
// Services return AppError; handlers map it to an RFC 7807 Problem Details response.
type AppError struct {
	// Code is the sentinel error (e.g. ErrNotFound) used for errors.Is() matching.
	Code error
	// Message is the safe, user-facing description. Must not contain internal details.
	Message string
	// HTTPStatus is the appropriate HTTP status code for this error.
	HTTPStatus int
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap allows errors.Is() to match against the wrapped sentinel.
func (e *AppError) Unwrap() error {
	return e.Code
}

// NewNotFound returns an AppError for a missing resource.
func NewNotFound(resource string) *AppError {
	return &AppError{
		Code:       ErrNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewForbidden returns an AppError for an ownership/permission violation.
func NewForbidden() *AppError {
	return &AppError{
		Code:       ErrForbidden,
		Message:    "you do not have permission to access this resource",
		HTTPStatus: http.StatusForbidden,
	}
}

// NewConflict returns an AppError for a business rule conflict.
func NewConflict(msg string) *AppError {
	return &AppError{
		Code:       ErrConflict,
		Message:    msg,
		HTTPStatus: http.StatusConflict,
	}
}

// NewValidation returns an AppError for input that fails business rule validation.
func NewValidation(msg string) *AppError {
	return &AppError{
		Code:       ErrValidation,
		Message:    msg,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

// NewInternal returns a generic 500 AppError. The internal detail is logged
// separately — never included in the response.
func NewInternal() *AppError {
	return &AppError{
		Code:       errors.New("internal"),
		Message:    "an unexpected error occurred",
		HTTPStatus: http.StatusInternalServerError,
	}
}
