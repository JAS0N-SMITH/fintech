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
)

// --- mock service ---

type mockPortfolioService struct {
	createFn  func(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error)
	getByIDFn func(ctx context.Context, callerID, portfolioID string) (*model.Portfolio, error)
	listFn    func(ctx context.Context, userID string) ([]*model.Portfolio, error)
	updateFn  func(ctx context.Context, callerID, portfolioID string, in model.UpdatePortfolioInput) (*model.Portfolio, error)
	deleteFn  func(ctx context.Context, callerID, portfolioID string) error
}

func (m *mockPortfolioService) Create(ctx context.Context, userID string, in model.CreatePortfolioInput) (*model.Portfolio, error) {
	return m.createFn(ctx, userID, in)
}
func (m *mockPortfolioService) GetByID(ctx context.Context, callerID, portfolioID string) (*model.Portfolio, error) {
	return m.getByIDFn(ctx, callerID, portfolioID)
}
func (m *mockPortfolioService) List(ctx context.Context, userID string) ([]*model.Portfolio, error) {
	return m.listFn(ctx, userID)
}
func (m *mockPortfolioService) Update(ctx context.Context, callerID, portfolioID string, in model.UpdatePortfolioInput) (*model.Portfolio, error) {
	return m.updateFn(ctx, callerID, portfolioID, in)
}
func (m *mockPortfolioService) Delete(ctx context.Context, callerID, portfolioID string) error {
	return m.deleteFn(ctx, callerID, portfolioID)
}

// --- test router helpers ---

// portfolioRouter builds a test engine with the given service and a stub auth
// middleware that injects callerID into the Gin context.
func portfolioRouter(svc *mockPortfolioService, callerID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Stub auth: set user_id directly so we don't need a real JWT.
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), callerID)
		c.Next()
	})

	h := NewPortfolioHandler(svc)
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func fixedPortfolio(id, userID, name string) *model.Portfolio {
	return &model.Portfolio{
		ID:        id,
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// --- List ---

func TestPortfolioHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		svcReturn  []*model.Portfolio
		svcErr     error
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns portfolios for authenticated user",
			svcReturn: []*model.Portfolio{
				fixedPortfolio("p1", "u1", "Main"),
				fixedPortfolio("p2", "u1", "Roth"),
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "empty list returns empty array",
			svcReturn:  []*model.Portfolio{},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "service error returns 500",
			svcErr:     errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPortfolioService{
				listFn: func(_ context.Context, _ string) ([]*model.Portfolio, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/portfolios", nil)
			w := httptest.NewRecorder()
			portfolioRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.svcErr == nil {
				var ps []*model.Portfolio
				if err := json.NewDecoder(w.Body).Decode(&ps); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(ps) != tt.wantLen {
					t.Errorf("len = %d, want %d", len(ps), tt.wantLen)
				}
			}
		})
	}
}

// --- Create ---

func TestPortfolioHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		svcReturn  *model.Portfolio
		svcErr     error
		wantStatus int
		wantName   string
	}{
		{
			name:       "valid input creates portfolio",
			body:       map[string]any{"name": "Brokerage"},
			svcReturn:  fixedPortfolio("p1", "u1", "Brokerage"),
			wantStatus: http.StatusCreated,
			wantName:   "Brokerage",
		},
		{
			name:       "missing name returns 400",
			body:       map[string]any{"description": "no name here"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service validation error returns 422",
			body:       map[string]any{"name": "Valid"},
			svcErr:     model.NewValidation("name too long"),
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "malformed JSON returns 400",
			body:       "{not json}",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPortfolioService{
				createFn: func(_ context.Context, _ string, _ model.CreatePortfolioInput) (*model.Portfolio, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			var bodyBytes []byte
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/portfolios", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			portfolioRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantName != "" {
				var p model.Portfolio
				if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if p.Name != tt.wantName {
					t.Errorf("name = %q, want %q", p.Name, tt.wantName)
				}
			}
		})
	}
}

// --- GetByID ---

func TestPortfolioHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		portfolioID string
		svcReturn  *model.Portfolio
		svcErr     error
		wantStatus int
	}{
		{
			name:        "owner gets portfolio",
			portfolioID: "p1",
			svcReturn:   fixedPortfolio("p1", "u1", "Main"),
			wantStatus:  http.StatusOK,
		},
		{
			name:        "not found returns 404",
			portfolioID: "missing",
			svcErr:      model.NewNotFound("portfolio"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "non-owner returns 403",
			portfolioID: "p2",
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPortfolioService{
				getByIDFn: func(_ context.Context, _, _ string) (*model.Portfolio, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/portfolios/"+tt.portfolioID, nil)
			w := httptest.NewRecorder()
			portfolioRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- Update ---

func TestPortfolioHandler_Update(t *testing.T) {
	tests := []struct {
		name        string
		portfolioID string
		body        any
		svcReturn   *model.Portfolio
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "owner updates portfolio",
			portfolioID: "p1",
			body:        map[string]any{"name": "Updated"},
			svcReturn:   fixedPortfolio("p1", "u1", "Updated"),
			wantStatus:  http.StatusOK,
		},
		{
			name:        "missing name returns 400",
			portfolioID: "p1",
			body:        map[string]any{"description": "no name"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "not found returns 404",
			portfolioID: "missing",
			body:        map[string]any{"name": "X"},
			svcErr:      model.NewNotFound("portfolio"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "non-owner returns 403",
			portfolioID: "p2",
			body:        map[string]any{"name": "X"},
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPortfolioService{
				updateFn: func(_ context.Context, _, _ string, _ model.UpdatePortfolioInput) (*model.Portfolio, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/portfolios/"+tt.portfolioID, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			portfolioRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- Delete ---

func TestPortfolioHandler_Delete(t *testing.T) {
	tests := []struct {
		name        string
		portfolioID string
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "owner deletes portfolio",
			portfolioID: "p1",
			wantStatus:  http.StatusNoContent,
		},
		{
			name:        "not found returns 404",
			portfolioID: "missing",
			svcErr:      model.NewNotFound("portfolio"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "non-owner returns 403",
			portfolioID: "p2",
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPortfolioService{
				deleteFn: func(_ context.Context, _, _ string) error {
					return tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodDelete, "/portfolios/"+tt.portfolioID, nil)
			w := httptest.NewRecorder()
			portfolioRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
