package handler

import (
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

type mockMarketDataService struct {
	getQuoteFn          func(ctx context.Context, symbol string) (*model.Quote, error)
	getHistoricalBarsFn func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error)
	searchSymbolsFn     func(ctx context.Context, query string, limit int) ([]model.Symbol, error)
}

func (m *mockMarketDataService) GetQuote(ctx context.Context, symbol string) (*model.Quote, error) {
	if m.getQuoteFn != nil {
		return m.getQuoteFn(ctx, symbol)
	}
	return &model.Quote{Symbol: symbol, Price: 100.0}, nil
}

func (m *mockMarketDataService) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	if m.getHistoricalBarsFn != nil {
		return m.getHistoricalBarsFn(ctx, symbol, tf, start, end)
	}
	return []model.Bar{}, nil
}

func (m *mockMarketDataService) SearchSymbols(ctx context.Context, query string, limit int) ([]model.Symbol, error) {
	if m.searchSymbolsFn != nil {
		return m.searchSymbolsFn(ctx, query, limit)
	}
	return []model.Symbol{}, nil
}

func marketDataRouter(svc *mockMarketDataService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserID), "user-1")
		c.Next()
	})
	h := NewMarketDataHandler(svc)
	h.RegisterRoutes(&r.RouterGroup)
	return r
}

// --- GetQuote tests ---

func TestMarketDataHandler_GetQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		symbol     string
		svcFn      func(ctx context.Context, symbol string) (*model.Quote, error)
		wantStatus int
		wantPrice  float64
	}{
		{
			name:       "valid symbol returns 200 with quote",
			symbol:     "AAPL",
			wantStatus: http.StatusOK,
			wantPrice:  100.0,
		},
		{
			name:       "lowercase symbol is uppercased and accepted",
			symbol:     "aapl",
			wantStatus: http.StatusOK,
			wantPrice:  100.0,
		},
		{
			name:       "invalid symbol (too long) returns 422",
			symbol:     "AAAAAAAAAAAAAAAAAAAAABBBBB",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "service error returns mapped status",
			symbol: "AAPL",
			svcFn: func(_ context.Context, _ string) (*model.Quote, error) {
				return nil, &model.AppError{
					Code:       model.ErrValidation,
					Message:    "symbol not found",
					HTTPStatus: http.StatusUnprocessableEntity,
				}
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "service 502 is returned",
			symbol: "AAPL",
			svcFn: func(_ context.Context, _ string) (*model.Quote, error) {
				return nil, &model.AppError{
					Code:       errors.New("provider"),
					Message:    "unavailable",
					HTTPStatus: http.StatusBadGateway,
				}
			},
			wantStatus: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &mockMarketDataService{getQuoteFn: tt.svcFn}
			r := marketDataRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/quotes/"+tt.symbol, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var q model.Quote
				if err := json.NewDecoder(w.Body).Decode(&q); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if q.Price != tt.wantPrice {
					t.Errorf("Price = %v, want %v", q.Price, tt.wantPrice)
				}
			}
		})
	}
}

// --- GetQuotesBatch tests ---

func TestMarketDataHandler_GetQuotesBatch(t *testing.T) {
	t.Parallel()

	t.Run("missing symbols param returns 400", func(t *testing.T) {
		t.Parallel()
		r := marketDataRouter(&mockMarketDataService{})
		req := httptest.NewRequest(http.MethodGet, "/quotes", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("valid symbols returns map of quotes", func(t *testing.T) {
		t.Parallel()
		r := marketDataRouter(&mockMarketDataService{})
		req := httptest.NewRequest(http.MethodGet, "/quotes?symbols=AAPL,MSFT", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
		}
		var result map[string]*model.Quote
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if result["AAPL"] == nil {
			t.Error("expected AAPL in result")
		}
		if result["MSFT"] == nil {
			t.Error("expected MSFT in result")
		}
	})

	t.Run("invalid symbols are skipped silently", func(t *testing.T) {
		t.Parallel()
		r := marketDataRouter(&mockMarketDataService{})
		req := httptest.NewRequest(http.MethodGet, "/quotes?symbols=AAPL,invalid-lower", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
		var result map[string]*model.Quote
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if result["AAPL"] == nil {
			t.Error("expected AAPL in result")
		}
	})
}

// --- GetBars tests ---

func TestMarketDataHandler_GetBars(t *testing.T) {
	t.Parallel()

	t.Run("valid request returns 200 with bars", func(t *testing.T) {
		t.Parallel()
		svc := &mockMarketDataService{
			getHistoricalBarsFn: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
				return []model.Bar{{Symbol: "AAPL", Close: 150.0}}, nil
			},
		}
		r := marketDataRouter(svc)
		req := httptest.NewRequest(http.MethodGet, "/bars/AAPL", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("invalid timeframe returns 422", func(t *testing.T) {
		t.Parallel()
		r := marketDataRouter(&mockMarketDataService{})
		req := httptest.NewRequest(http.MethodGet, "/bars/AAPL?timeframe=INVALID", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want 422", w.Code)
		}
	})

	t.Run("invalid start date returns 422", func(t *testing.T) {
		t.Parallel()
		r := marketDataRouter(&mockMarketDataService{})
		req := httptest.NewRequest(http.MethodGet, "/bars/AAPL?start=not-a-date", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want 422", w.Code)
		}
	})

	t.Run("end before start returns 422", func(t *testing.T) {
		t.Parallel()
		r := marketDataRouter(&mockMarketDataService{})
		req := httptest.NewRequest(http.MethodGet,
			"/bars/AAPL?start=2024-12-31T00:00:00Z&end=2024-01-01T00:00:00Z", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want 422", w.Code)
		}
	})
}
