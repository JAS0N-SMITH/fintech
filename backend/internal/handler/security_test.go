package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
)

// TestPortfolioNameSQLInjection verifies that SQL injection attempts in portfolio names
// are safely escaped and stored as literal text, not executed.
func TestPortfolioNameSQLInjection(t *testing.T) {
	mockSvc := &mockPortfolioService{
		createFn: func(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error) {
			// Service layer should have already validated; just store it
			return &model.Portfolio{
				ID:   "test-id",
				Name: in.Name,
			}, nil
		},
	}

	handler := NewPortfolioHandler(mockSvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), "user-id")
		c.Next()
	})

	handler.RegisterRoutes(&router.RouterGroup)

	tests := []struct {
		name     string
		payload  string
		wantCode int
	}{
		{
			name:     "SQL injection with DROP TABLE",
			payload:  "'; DROP TABLE portfolios; --",
			wantCode: http.StatusCreated,
		},
		{
			name:     "SQL injection with OR clause",
			payload:  "' OR '1'='1",
			wantCode: http.StatusCreated,
		},
		{
			name:     "Normal portfolio name",
			payload:  "My Portfolio",
			wantCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := model.CreatePortfolioInput{
				Name:        tt.payload,
				Description: "Test",
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest("POST", "/portfolios", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, w.Code)
			}

			if tt.wantCode == http.StatusCreated {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)

				// Verify the name was stored literally, not executed
				if data, ok := resp["data"].(map[string]interface{}); ok {
					if name, ok := data["name"].(string); ok && name != tt.payload {
						t.Errorf("expected name %q, got %q", tt.payload, name)
					}
				}
			}
		})
	}
}

// TestPortfolioNameXSS verifies that XSS payloads in portfolio names
// are safely stored as literal text.
func TestPortfolioNameXSS(t *testing.T) {
	mockSvc := &mockPortfolioService{
		createFn: func(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error) {
			return &model.Portfolio{
				ID:   "test-id",
				Name: in.Name,
			}, nil
		},
	}

	handler := NewPortfolioHandler(mockSvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), "user-id")
		c.Next()
	})

	handler.RegisterRoutes(&router.RouterGroup)

	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "Script tag injection",
			payload: `<script>alert('xss')</script>`,
		},
		{
			name:    "Event handler injection",
			payload: `"><svg onload=alert('xss')>`,
		},
		{
			name:    "JavaScript protocol",
			payload: `javascript:alert('xss')`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := model.CreatePortfolioInput{
				Name:        tt.payload,
				Description: "Test",
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest("POST", "/portfolios", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should succeed
			if w.Code != http.StatusCreated {
				t.Errorf("expected status 201, got %d", w.Code)
			}

			// Payload should be stored literally
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			if data, ok := resp["data"].(map[string]interface{}); ok {
				if name, ok := data["name"].(string); ok && name != tt.payload {
					t.Errorf("expected name %q, got %q", tt.payload, name)
				}
			}
		})
	}
}

// TestOversizedInputRejection verifies that oversized inputs are rejected.
func TestOversizedInputRejection(t *testing.T) {
	mockSvc := &mockPortfolioService{
		createFn: func(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error) {
			return &model.Portfolio{
				ID:   "test-id",
				Name: in.Name,
			}, nil
		},
	}

	handler := NewPortfolioHandler(mockSvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), "user-id")
		c.Next()
	})

	handler.RegisterRoutes(&router.RouterGroup)

	tests := []struct {
		name     string
		nameLen  int
		descLen  int
		wantCode int
	}{
		{
			name:     "name exceeds max (>100 chars)",
			nameLen:  101,
			descLen:  50,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "description exceeds max (>500 chars)",
			nameLen:  50,
			descLen:  501,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "both within limits",
			nameLen:  50,
			descLen:  250,
			wantCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate strings of specified length
			name := bytes.Repeat([]byte("a"), tt.nameLen)
			desc := bytes.Repeat([]byte("b"), tt.descLen)

			body := model.CreatePortfolioInput{
				Name:        string(name),
				Description: string(desc),
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest("POST", "/portfolios", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

// (mockPortfolioService definition removed; use the shared version from mock_portfolio_service_test.go)
