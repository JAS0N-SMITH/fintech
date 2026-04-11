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
	"github.com/shopspring/decimal"

	"github.com/huchknows/fintech/backend/internal/middleware"
	"github.com/huchknows/fintech/backend/internal/model"
)

// --- mock service ---

type mockTransactionService struct {
	createFn func(ctx context.Context, callerID, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error)
	listFn   func(ctx context.Context, callerID, portfolioID string) ([]*model.Transaction, error)
	deleteFn func(ctx context.Context, callerID, transactionID string) error
}

func (m *mockTransactionService) Create(ctx context.Context, callerID, portfolioID string, in model.CreateTransactionInput) (*model.Transaction, error) {
	return m.createFn(ctx, callerID, portfolioID, in)
}
func (m *mockTransactionService) List(ctx context.Context, callerID, portfolioID string) ([]*model.Transaction, error) {
	return m.listFn(ctx, callerID, portfolioID)
}
func (m *mockTransactionService) Delete(ctx context.Context, callerID, transactionID string) error {
	return m.deleteFn(ctx, callerID, transactionID)
}

// --- test router helpers ---

func transactionRouter(svc *mockTransactionService, callerID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), callerID)
		c.Next()
	})

	h := NewTransactionHandler(svc)
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

func fixedTransaction(id, portfolioID string, txType model.TransactionType) *model.Transaction {
	qty := decimal.NewFromInt(10)
	price := decimal.NewFromFloat(150.00)
	return &model.Transaction{
		ID:              id,
		PortfolioID:     portfolioID,
		TransactionType: txType,
		Symbol:          "AAPL",
		TransactionDate: time.Now(),
		Quantity:        &qty,
		PricePerShare:   &price,
		TotalAmount:     decimal.NewFromFloat(1500.00),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// --- List ---

func TestTransactionHandler_List(t *testing.T) {
	tests := []struct {
		name        string
		portfolioID string
		svcReturn   []*model.Transaction
		svcErr      error
		wantStatus  int
		wantLen     int
	}{
		{
			name:        "owner lists transactions",
			portfolioID: "p1",
			svcReturn: []*model.Transaction{
				fixedTransaction("t1", "p1", model.TransactionTypeBuy),
				fixedTransaction("t2", "p1", model.TransactionTypeSell),
			},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:        "empty portfolio returns empty array",
			portfolioID: "p1",
			svcReturn:   []*model.Transaction{},
			wantStatus:  http.StatusOK,
			wantLen:     0,
		},
		{
			name:        "non-owner portfolio returns 403",
			portfolioID: "p2",
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "unknown portfolio returns 404",
			portfolioID: "missing",
			svcErr:      model.NewNotFound("portfolio"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "service error returns 500",
			portfolioID: "p1",
			svcErr:      errors.New("db down"),
			wantStatus:  http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTransactionService{
				listFn: func(_ context.Context, _, _ string) ([]*model.Transaction, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/portfolios/"+tt.portfolioID+"/transactions", nil)
			w := httptest.NewRecorder()
			transactionRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.svcErr == nil {
				var txns []*model.Transaction
				if err := json.NewDecoder(w.Body).Decode(&txns); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(txns) != tt.wantLen {
					t.Errorf("len = %d, want %d", len(txns), tt.wantLen)
				}
			}
		})
	}
}

// --- Create ---

func TestTransactionHandler_Create(t *testing.T) {
	validBuyBody := map[string]any{
		"transaction_type": "buy",
		"symbol":           "AAPL",
		"transaction_date": "2024-01-15",
		"quantity":         "10",
		"price_per_share":  "150.00",
		"total_amount":     "1500.00",
	}

	tests := []struct {
		name        string
		portfolioID string
		body        any
		svcReturn   *model.Transaction
		svcErr      error
		wantStatus  int
	}{
		{
			name:        "valid buy transaction created",
			portfolioID: "p1",
			body:        validBuyBody,
			svcReturn:   fixedTransaction("t1", "p1", model.TransactionTypeBuy),
			wantStatus:  http.StatusCreated,
		},
		{
			name:        "missing transaction_type returns 400",
			portfolioID: "p1",
			body:        map[string]any{"symbol": "AAPL", "transaction_date": "2024-01-15", "total_amount": "100"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "missing symbol returns 400",
			portfolioID: "p1",
			body:        map[string]any{"transaction_type": "buy", "transaction_date": "2024-01-15", "total_amount": "100"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "service validation error returns 422",
			portfolioID: "p1",
			body:        validBuyBody,
			svcErr:      model.NewValidation("quantity is required for buy"),
			wantStatus:  http.StatusUnprocessableEntity,
		},
		{
			name:        "insufficient holdings returns 409",
			portfolioID: "p1",
			body:        validBuyBody,
			svcErr:      model.NewConflict("cannot sell more than held"),
			wantStatus:  http.StatusConflict,
		},
		{
			name:        "portfolio not found returns 404",
			portfolioID: "missing",
			body:        validBuyBody,
			svcErr:      model.NewNotFound("portfolio"),
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "non-owner portfolio returns 403",
			portfolioID: "p2",
			body:        validBuyBody,
			svcErr:      model.NewForbidden(),
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "malformed JSON returns 400",
			portfolioID: "p1",
			body:        "{not json}",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTransactionService{
				createFn: func(_ context.Context, _, _ string, _ model.CreateTransactionInput) (*model.Transaction, error) {
					return tt.svcReturn, tt.svcErr
				},
			}

			var bodyBytes []byte
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/portfolios/"+tt.portfolioID+"/transactions", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			transactionRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- Delete ---

func TestTransactionHandler_Delete(t *testing.T) {
	tests := []struct {
		name          string
		portfolioID   string
		transactionID string
		svcErr        error
		wantStatus    int
	}{
		{
			name:          "owner deletes transaction",
			portfolioID:   "p1",
			transactionID: "t1",
			wantStatus:    http.StatusNoContent,
		},
		{
			name:          "transaction not found returns 404",
			portfolioID:   "p1",
			transactionID: "missing",
			svcErr:        model.NewNotFound("transaction"),
			wantStatus:    http.StatusNotFound,
		},
		{
			name:          "non-owner portfolio returns 403",
			portfolioID:   "p2",
			transactionID: "t1",
			svcErr:        model.NewForbidden(),
			wantStatus:    http.StatusForbidden,
		},
		{
			name:          "service error returns 500",
			portfolioID:   "p1",
			transactionID: "t1",
			svcErr:        errors.New("db error"),
			wantStatus:    http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTransactionService{
				deleteFn: func(_ context.Context, _, _ string) error {
					return tt.svcErr
				},
			}

			url := "/portfolios/" + tt.portfolioID + "/transactions/" + tt.transactionID
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			w := httptest.NewRecorder()
			transactionRouter(svc, "u1").ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
