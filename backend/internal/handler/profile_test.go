package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
)

// --- mock profile service ---

type mockProfileService struct {
	getByIDFn         func(ctx context.Context, id string) (*model.UserProfile, error)
	updatePreferenceFn func(ctx context.Context, id string, preferences json.RawMessage) error
}

func (m *mockProfileService) GetByID(ctx context.Context, id string) (*model.UserProfile, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &model.UserProfile{ID: id}, nil
}

func (m *mockProfileService) UpdatePreferences(ctx context.Context, id string, preferences json.RawMessage) error {
	if m.updatePreferenceFn != nil {
		return m.updatePreferenceFn(ctx, id, preferences)
	}
	return nil
}

// --- test router helper ---

func profileRouter(svc *mockProfileService, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Stub auth: set user_id directly so we don't need a real JWT.
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), userID)
		c.Next()
	})

	h := NewProfileHandler(svc)
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func fixedProfile(id, displayName string) *model.UserProfile {
	return &model.UserProfile{
		ID:          id,
		DisplayName: displayName,
		Role:        "user",
		Preferences: json.RawMessage(`{"alert_thresholds":[]}`),
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-01T00:00:00Z",
	}
}

// --- GetMe ---

func TestProfileHandler_GetMe(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		svcReturn  *model.UserProfile
		svcErr     error
		wantStatus int
	}{
		{
			name:       "returns authenticated user's profile",
			userID:     "u1",
			svcReturn:  fixedProfile("u1", "Alice"),
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found returns 404",
			userID:     "missing",
			svcErr:     model.NewNotFound("profile"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "service error returns 500",
			userID:     "u1",
			svcErr:     errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockProfileService{
				getByIDFn: func(_ context.Context, _ string) (*model.UserProfile, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			w := httptest.NewRecorder()
			profileRouter(svc, tt.userID).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.svcErr == nil {
				var p model.UserProfile
				if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if p.ID != tt.userID {
					t.Errorf("profile.ID = %q, want %q", p.ID, tt.userID)
				}
			}
		})
	}
}

// --- UpdatePreferences ---

func TestProfileHandler_UpdatePreferences(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		svcErr     error
		wantStatus int
	}{
		{
			name:       "valid alert thresholds returns 204",
			body:       map[string]any{"alert_thresholds": json.RawMessage(`[5,10]`)},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "empty alert thresholds returns 204",
			body:       map[string]any{"alert_thresholds": json.RawMessage(`[]`)},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "malformed JSON returns 400",
			body:       "{bad json}",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing alert_thresholds (null) returns 204",
			body:       map[string]any{"alert_thresholds": nil},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "service error returns 500",
			body:       map[string]any{"alert_thresholds": json.RawMessage(`[]`)},
			svcErr:     errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockProfileService{
				updatePreferenceFn: func(_ context.Context, _ string, _ json.RawMessage) error {
					return tt.svcErr
				},
			}

			var bodyBytes []byte
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPatch, "/me/preferences", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			profileRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
