package model

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	err := &AppError{
		Code:       ErrNotFound,
		Message:    "portfolio not found",
		HTTPStatus: http.StatusNotFound,
	}

	expected := "not found: portfolio not found"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestAppError_Unwrap(t *testing.T) {
	err := &AppError{
		Code:       ErrNotFound,
		Message:    "user not found",
		HTTPStatus: http.StatusNotFound,
	}

	if !errors.Is(err, ErrNotFound) {
		t.Error("errors.Is(appErr, ErrNotFound) = false, want true")
	}
}

func TestNewNotFound(t *testing.T) {
	err := NewNotFound("transaction")
	if err.Code != ErrNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrNotFound)
	}
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusNotFound)
	}
	if err.Message != "transaction not found" {
		t.Errorf("Message = %q, want %q", err.Message, "transaction not found")
	}
}

func TestNewForbidden(t *testing.T) {
	err := NewForbidden()
	if err.Code != ErrForbidden {
		t.Errorf("Code = %v, want %v", err.Code, ErrForbidden)
	}
	if err.HTTPStatus != http.StatusForbidden {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusForbidden)
	}
	if err.Message != "you do not have permission to access this resource" {
		t.Errorf("Message = %q, unexpected", err.Message)
	}
}

func TestNewConflict(t *testing.T) {
	err := NewConflict("cannot sell more shares than owned")
	if err.Code != ErrConflict {
		t.Errorf("Code = %v, want %v", err.Code, ErrConflict)
	}
	if err.HTTPStatus != http.StatusConflict {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusConflict)
	}
	if err.Message != "cannot sell more shares than owned" {
		t.Errorf("Message = %q, want %q", err.Message, "cannot sell more shares than owned")
	}
}

func TestNewValidation(t *testing.T) {
	err := NewValidation("quantity must be positive")
	if err.Code != ErrValidation {
		t.Errorf("Code = %v, want %v", err.Code, ErrValidation)
	}
	if err.HTTPStatus != http.StatusUnprocessableEntity {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusUnprocessableEntity)
	}
	if err.Message != "quantity must be positive" {
		t.Errorf("Message = %q, want %q", err.Message, "quantity must be positive")
	}
}

func TestNewInternal(t *testing.T) {
	err := NewInternal()
	if err.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusInternalServerError)
	}
	if err.Message != "an unexpected error occurred" {
		t.Errorf("Message = %q, want %q", err.Message, "an unexpected error occurred")
	}
}
