package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/model"
)

func problemRouter(err error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		RespondError(c, err)
	})
	return r
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantDetail string
	}{
		{
			name:       "ErrNotFound AppError maps to 404",
			err:        model.NewNotFound("portfolio"),
			wantStatus: http.StatusNotFound,
			wantDetail: "portfolio not found",
		},
		{
			name:       "ErrForbidden AppError maps to 403",
			err:        model.NewForbidden(),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "ErrConflict AppError maps to 409",
			err:        model.NewConflict("cannot sell more than held"),
			wantStatus: http.StatusConflict,
			wantDetail: "cannot sell more than held",
		},
		{
			name:       "ErrValidation AppError maps to 422",
			err:        model.NewValidation("quantity is required"),
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "quantity is required",
		},
		{
			name:       "wrapped AppError is matched via errors.As",
			err:        errors.Join(errors.New("wrap"), model.NewNotFound("transaction")),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown error maps to 500",
			err:        errors.New("db connection lost"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			problemRouter(tt.err).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var p Problem
			if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if p.Status != tt.wantStatus {
				t.Errorf("body.status = %d, want %d", p.Status, tt.wantStatus)
			}
			if tt.wantDetail != "" && p.Detail != tt.wantDetail {
				t.Errorf("body.detail = %q, want %q", p.Detail, tt.wantDetail)
			}
		})
	}
}
