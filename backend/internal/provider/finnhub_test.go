package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// newTestProvider creates a FinnhubProvider pointed at a test server.
func newTestProvider(server *httptest.Server) *FinnhubProvider {
	return NewFinnhubProvider("test-key", server.URL, "")
}

// --- GetQuote tests ---

func TestFinnhubProvider_GetQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		symbol    string
		wantErr   error
		wantPrice float64
		wantHigh  float64
	}{
		{
			name:   "valid symbol returns populated quote",
			symbol: "AAPL",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(finnhubQuoteResponse{
					CurrentPrice:  150.25,
					DayHigh:       152.00,
					DayLow:        148.50,
					Open:          149.00,
					PreviousClose: 148.00,
					Volume:        1000000,
					Timestamp:     1700000000,
				})
			},
			wantPrice: 150.25,
			wantHigh:  152.00,
		},
		{
			name:   "zero price and zero timestamp returns ErrInvalidSymbol",
			symbol: "INVALID",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// Finnhub returns all zeros for unknown symbols
				_ = json.NewEncoder(w).Encode(finnhubQuoteResponse{})
			},
			wantErr: ErrInvalidSymbol,
		},
		{
			name:   "HTTP 429 returns ErrRateLimited",
			symbol: "AAPL",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			wantErr: ErrRateLimited,
		},
		{
			name:   "HTTP 403 returns ErrInvalidSymbol",
			symbol: "BADKEY",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			wantErr: ErrInvalidSymbol,
		},
		{
			name:   "HTTP 500 returns ErrProviderUnavailable",
			symbol: "AAPL",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: ErrProviderUnavailable,
		},
		{
			name:   "malformed JSON returns ErrProviderUnavailable",
			symbol: "AAPL",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{not valid json`))
			},
			wantErr: ErrProviderUnavailable,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.handler)
			defer server.Close()

			p := newTestProvider(server)
			quote, err := p.GetQuote(context.Background(), tt.symbol)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetQuote() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetQuote() unexpected error: %v", err)
			}
			if quote.Symbol != tt.symbol {
				t.Errorf("Symbol = %q, want %q", quote.Symbol, tt.symbol)
			}
			if quote.Price != tt.wantPrice {
				t.Errorf("Price = %v, want %v", quote.Price, tt.wantPrice)
			}
			if quote.DayHigh != tt.wantHigh {
				t.Errorf("DayHigh = %v, want %v", quote.DayHigh, tt.wantHigh)
			}
		})
	}
}

// --- GetHistoricalBars tests ---

func TestFinnhubProvider_GetHistoricalBars(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		timeframe model.Timeframe
		wantErr   error
		wantLen   int
	}{
		{
			name:      "valid response returns bars",
			timeframe: model.Timeframe1M,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(finnhubCandleResponse{
					Close:     []float64{150.0, 151.0},
					High:      []float64{152.0, 153.0},
					Low:       []float64{148.0, 149.0},
					Open:      []float64{149.0, 150.0},
					Timestamp: []int64{1700000000, 1700086400},
					Volume:    []int64{1000000, 1100000},
					Status:    "ok",
				})
			},
			wantLen: 2,
		},
		{
			name:      "no_data status returns empty slice",
			timeframe: model.Timeframe1M,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(finnhubCandleResponse{Status: "no_data"})
			},
			wantLen: 0,
		},
		{
			name:      "HTTP 429 returns ErrRateLimited",
			timeframe: model.Timeframe1M,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			wantErr: ErrRateLimited,
		},
		{
			name:      "unknown timeframe returns ErrInvalidSymbol",
			timeframe: model.Timeframe("INVALID"),
			handler: func(w http.ResponseWriter, _ *http.Request) {
				// Should never be called for unknown timeframe
				w.WriteHeader(http.StatusOK)
			},
			wantErr: ErrInvalidSymbol,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.handler)
			defer server.Close()

			p := newTestProvider(server)
			bars, err := p.GetHistoricalBars(context.Background(), "AAPL", tt.timeframe, start, end)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetHistoricalBars() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetHistoricalBars() unexpected error: %v", err)
			}
			if len(bars) != tt.wantLen {
				t.Errorf("len(bars) = %d, want %d", len(bars), tt.wantLen)
			}
			if tt.wantLen > 0 {
				if bars[0].Close != 150.0 {
					t.Errorf("bars[0].Close = %v, want 150.0", bars[0].Close)
				}
				if bars[0].Symbol != "AAPL" {
					t.Errorf("bars[0].Symbol = %q, want AAPL", bars[0].Symbol)
				}
			}
		})
	}
}

// --- MockProvider tests ---

func TestMockProvider_GetQuote(t *testing.T) {
	t.Parallel()

	t.Run("uses GetQuoteFn when set", func(t *testing.T) {
		m := &MockProvider{
			GetQuoteFn: func(_ context.Context, symbol string) (*model.Quote, error) {
				return &model.Quote{Symbol: symbol, Price: 99.0}, nil
			},
		}
		q, err := m.GetQuote(context.Background(), "MSFT")
		if err != nil {
			t.Fatal(err)
		}
		if q.Price != 99.0 {
			t.Errorf("Price = %v, want 99.0", q.Price)
		}
	})

	t.Run("returns zero Quote when Fn not set", func(t *testing.T) {
		m := &MockProvider{}
		q, err := m.GetQuote(context.Background(), "MSFT")
		if err != nil {
			t.Fatal(err)
		}
		if q.Symbol != "MSFT" {
			t.Errorf("Symbol = %q, want MSFT", q.Symbol)
		}
	})
}

func TestMockProvider_GetHistoricalBars(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when Fn not set", func(t *testing.T) {
		m := &MockProvider{}
		bars, err := m.GetHistoricalBars(context.Background(), "AAPL", model.Timeframe1M, time.Now(), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if len(bars) != 0 {
			t.Errorf("len(bars) = %d, want 0", len(bars))
		}
	})
}
