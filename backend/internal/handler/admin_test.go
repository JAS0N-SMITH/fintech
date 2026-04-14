package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/repository"
)

// --- mock service ---

type mockAdminService struct {
	listUsersFn      func(ctx context.Context, page, pageSize int) (*model.AdminUserList, error)
	updateUserRoleFn func(ctx context.Context, adminID, targetID, newRole string) (*model.AdminUser, error)
	listAuditLogFn   func(ctx context.Context, filter repository.AuditLogFilter) (*model.AuditLogList, error)
	recordAuditFn    func(ctx context.Context, entry model.AuditLogEntry) error
	getHealthFn      func(ctx context.Context) (*model.HealthStatus, error)
}

func (m *mockAdminService) ListUsers(ctx context.Context, page, pageSize int) (*model.AdminUserList, error) {
	return m.listUsersFn(ctx, page, pageSize)
}

func (m *mockAdminService) UpdateUserRole(ctx context.Context, adminID, targetID, newRole string) (*model.AdminUser, error) {
	return m.updateUserRoleFn(ctx, adminID, targetID, newRole)
}

func (m *mockAdminService) ListAuditLog(ctx context.Context, filter repository.AuditLogFilter) (*model.AuditLogList, error) {
	return m.listAuditLogFn(ctx, filter)
}

func (m *mockAdminService) RecordAuditEvent(ctx context.Context, entry model.AuditLogEntry) error {
	return m.recordAuditFn(ctx, entry)
}

func (m *mockAdminService) GetSystemHealth(ctx context.Context) (*model.HealthStatus, error) {
	return m.getHealthFn(ctx)
}

// --- test router helpers ---

// adminRouter builds a test engine with the given service and a stub auth
// middleware that injects userID and role into the Gin context.
func adminRouter(svc *mockAdminService, userID, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Stub auth: set user_id and user_role directly so we don't need a real JWT.
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), userID)
		c.Set(string(middleware.ContextKeyUserRole), role)
		c.Next()
	})

	h := NewAdminHandler(svc)
	adminGroup := r.Group("/admin")
	h.RegisterRoutes(adminGroup)
	return r
}

func fixedAdminUser(id, email, displayName, role string) *model.AdminUser {
	return &model.AdminUser{
		ID:          id,
		Email:       email,
		DisplayName: displayName,
		Role:        role,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// --- ListUsers ---

func TestAdminHandler_ListUsers(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		svcReturn  *model.AdminUserList
		svcErr     error
		wantStatus int
		wantLen    int
	}{
		{
			name:  "returns users with default pagination",
			query: "",
			svcReturn: &model.AdminUserList{
				Users: []model.AdminUser{
					*fixedAdminUser("u1", "alice@example.com", "Alice", "user"),
					*fixedAdminUser("u2", "bob@example.com", "Bob", "admin"),
				},
				Total: 2,
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:  "respects page and page_size query params",
			query: "?page=2&page_size=10",
			svcReturn: &model.AdminUserList{
				Users: []model.AdminUser{},
				Total: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "service error returns 500",
			query:      "",
			svcErr:     errors.New("db error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAdminService{
				listUsersFn: func(_ context.Context, _, _ int) (*model.AdminUserList, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/admin/users"+tt.query, nil)
			w := httptest.NewRecorder()
			adminRouter(svc, "admin-id", "admin").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.svcErr == nil && tt.wantLen > 0 {
				var list model.AdminUserList
				if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(list.Users) != tt.wantLen {
					t.Errorf("len = %d, want %d", len(list.Users), tt.wantLen)
				}
			}
		})
	}
}

// --- PatchRole ---

func TestAdminHandler_PatchRole(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		body       any
		svcReturn  *model.AdminUser
		svcErr     error
		wantStatus int
		wantRole   string
	}{
		{
			name:       "valid role change returns 200",
			userID:     "target-user",
			body:       map[string]any{"role": "admin"},
			svcReturn:  fixedAdminUser("target-user", "target@example.com", "Target", "admin"),
			wantStatus: http.StatusOK,
			wantRole:   "admin",
		},
		{
			name:       "missing role returns 400",
			userID:     "target-user",
			body:       map[string]any{"other": "field"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid role value returns 400",
			userID:     "target-user",
			body:       map[string]any{"role": "superadmin"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "self-role-change returns 409 conflict",
			userID:     "admin-id",
			body:       map[string]any{"role": "user"},
			svcErr:     model.NewConflict("cannot change your own role"),
			wantStatus: http.StatusConflict,
		},
		{
			name:       "not found returns 404",
			userID:     "nonexistent",
			body:       map[string]any{"role": "admin"},
			svcErr:     model.NewNotFound("user"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "malformed JSON returns 400",
			userID:     "target-user",
			body:       "{invalid json}",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAdminService{
				updateUserRoleFn: func(_ context.Context, _, _, _ string) (*model.AdminUser, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			var bodyBytes []byte
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPatch, "/admin/users/"+tt.userID+"/role", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			adminRouter(svc, "admin-id", "admin").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantRole != "" {
				var u model.AdminUser
				if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if u.Role != tt.wantRole {
					t.Errorf("role = %q, want %q", u.Role, tt.wantRole)
				}
			}
		})
	}
}

// --- ListAuditLog ---

func TestAdminHandler_ListAuditLog(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		svcReturn  *model.AuditLogList
		svcErr     error
		wantStatus int
		wantLen    int
	}{
		{
			name:  "returns audit log with default pagination",
			query: "",
			svcReturn: &model.AuditLogList{
				Entries: []model.AuditLogEntry{
					{
						ID:           "log1",
						UserID:       "admin-id",
						Action:       "role_change",
						TargetEntity: "user",
						TargetID:     "target-id",
						IPAddress:    "127.0.0.1",
						UserAgent:    "test",
						CreatedAt:    time.Now(),
					},
				},
				Total:    1,
				Page:     1,
				PageSize: 25,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "service error returns 500",
			query:      "",
			svcErr:     errors.New("db error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAdminService{
				listAuditLogFn: func(_ context.Context, _ repository.AuditLogFilter) (*model.AuditLogList, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/admin/audit-log"+tt.query, nil)
			w := httptest.NewRecorder()
			adminRouter(svc, "admin-id", "admin").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.svcErr == nil && tt.wantLen > 0 {
				var list model.AuditLogList
				if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(list.Entries) != tt.wantLen {
					t.Errorf("len = %d, want %d", len(list.Entries), tt.wantLen)
				}
			}
		})
	}
}

// --- Health ---

func TestAdminHandler_Health(t *testing.T) {
	tests := []struct {
		name       string
		svcReturn  *model.HealthStatus
		svcErr     error
		wantStatus int
	}{
		{
			name: "all systems healthy returns 200",
			svcReturn: &model.HealthStatus{
				DB:             "healthy",
				FinnhubAPI:     "healthy",
				WebSocketCount: 5,
				Timestamp:      time.Now(),
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "degraded system returns 200",
			svcReturn: &model.HealthStatus{
				DB:             "healthy",
				FinnhubAPI:     "unhealthy",
				WebSocketCount: 0,
				Timestamp:      time.Now(),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "service error returns 500",
			svcErr:     errors.New("failed to check health"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAdminService{
				getHealthFn: func(_ context.Context) (*model.HealthStatus, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/admin/health", nil)
			w := httptest.NewRecorder()
			adminRouter(svc, "admin-id", "admin").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
