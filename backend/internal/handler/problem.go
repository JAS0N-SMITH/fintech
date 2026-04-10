// Package handler contains HTTP handlers that parse requests and return responses.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
)

// Problem is an RFC 7807 Problem Details response body.
// See https://www.rfc-editor.org/rfc/rfc7807
type Problem struct {
	// Status mirrors the HTTP status code.
	Status int `json:"status"`
	// Title is a short, human-readable summary of the problem type.
	Title string `json:"title"`
	// Detail is a human-readable explanation specific to this occurrence.
	Detail string `json:"detail,omitempty"`
	// Instance is an optional URI identifying this specific occurrence.
	Instance string `json:"instance,omitempty"`
}

// RespondError maps an error to an RFC 7807 response and writes it.
// AppErrors are mapped to their HTTPStatus; unknown errors become 500.
// Internal details are logged but never sent to the client.
func RespondError(c *gin.Context, err error) {
	var appErr *model.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, Problem{
			Status: appErr.HTTPStatus,
			Title:  http.StatusText(appErr.HTTPStatus),
			Detail: appErr.Message,
		})
		return
	}

	// Unknown error — log full detail internally, return generic 500.
	slog.ErrorContext(c.Request.Context(), "unhandled error",
		"request_id", middleware.RequestIDFromContext(c),
		"error", err,
	)
	c.JSON(http.StatusInternalServerError, Problem{
		Status: http.StatusInternalServerError,
		Title:  http.StatusText(http.StatusInternalServerError),
		Detail: "an unexpected error occurred",
	})
}
