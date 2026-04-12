package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/model"
)

// mockAuditService mocks the AdminService for testing.
type mockAuditService struct {
	recordFn func(ctx context.Context, entry model.AuditLogEntry) error
}

func (m *mockAuditService) RecordAuditEvent(ctx context.Context, entry model.AuditLogEntry) error {
	if m.recordFn != nil {
		return m.recordFn(ctx, entry)
	}
	return nil
}

func (m *mockAuditService) ListUsers(ctx context.Context, page, pageSize int) (*model.AdminUserList, error) {
	return nil, nil
}

func (m *mockAuditService) UpdateUserRole(ctx context.Context, adminID, targetID, newRole string) (*model.AdminUser, error) {
	return nil, nil
}

func (m *mockAuditService) ListAuditLog(ctx context.Context, page, pageSize int) (*model.AuditLogList, error) {
	return nil, nil
}

func (m *mockAuditService) GetSystemHealth(ctx context.Context) (*model.HealthStatus, error) {
	return nil, nil
}

// TestAuditAction_RecordsOnSuccess verifies that RecordAuditEvent is called on 2xx responses.
func TestAuditAction_RecordsOnSuccess(t *testing.T) {
	recordCalled := false

	svc := &mockAuditService{
		recordFn: func(_ context.Context, entry model.AuditLogEntry) error {
			recordCalled = true
			if entry.Action != "test_action" {
				t.Errorf("action = %q, want test_action", entry.Action)
			}
			if entry.TargetEntity != "test_entity" {
				t.Errorf("target_entity = %q, want test_entity", entry.TargetEntity)
			}
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Set user context
	r.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUserID), "user-123")
		c.Next()
	})

	// Test endpoint with audit middleware
	r.GET("/test", AuditAction("test_action", "test_entity", svc), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if !recordCalled {
		t.Error("RecordAuditEvent was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// TestAuditAction_SkipsOnError verifies that RecordAuditEvent is not called on 4xx/5xx responses.
func TestAuditAction_SkipsOnError(t *testing.T) {
	recordCalled := false

	svc := &mockAuditService{
		recordFn: func(_ context.Context, entry model.AuditLogEntry) error {
			recordCalled = true
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUserID), "user-123")
		c.Next()
	})

	r.GET("/test", AuditAction("test_action", "test_entity", svc), func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if recordCalled {
		t.Error("RecordAuditEvent should not be called for 4xx response")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestAuditAction_SkipsOnServerError verifies that RecordAuditEvent is not called on 5xx responses.
func TestAuditAction_SkipsOnServerError(t *testing.T) {
	recordCalled := false

	svc := &mockAuditService{
		recordFn: func(_ context.Context, entry model.AuditLogEntry) error {
			recordCalled = true
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUserID), "user-123")
		c.Next()
	})

	r.GET("/test", AuditAction("test_action", "test_entity", svc), func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oops"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if recordCalled {
		t.Error("RecordAuditEvent should not be called for 5xx response")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// TestAuditAction_CapturesIPAndUserAgent verifies that IP and User-Agent are captured.
func TestAuditAction_CapturesIPAndUserAgent(t *testing.T) {
	var capturedEntry model.AuditLogEntry

	svc := &mockAuditService{
		recordFn: func(_ context.Context, entry model.AuditLogEntry) error {
			capturedEntry = entry
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUserID), "user-123")
		c.Next()
	})

	r.GET("/test", AuditAction("test_action", "test_entity", svc), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "TestClient/1.0")
	req.RemoteAddr = "192.168.1.100:12345"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedEntry.UserAgent != "TestClient/1.0" {
		t.Errorf("user_agent = %q, want TestClient/1.0", capturedEntry.UserAgent)
	}
	// Note: ClientIP() in test context returns request.RemoteAddr or 127.0.0.1
	if capturedEntry.IPAddress == "" {
		t.Error("IP address was not captured")
	}
}
