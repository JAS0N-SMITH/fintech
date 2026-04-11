package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/provider"
)

// --- helpers ---

func fixedQuote(symbol string, price float64) *model.Quote {
	return &model.Quote{
		Symbol:        symbol,
		Price:         price,
		DayHigh:       price + 2,
		DayLow:        price - 2,
		Open:          price - 1,
		PreviousClose: price - 0.5,
		Volume:        1000000,
		Timestamp:     time.Now().UTC(),
	}
}

func fixedBars(symbol string, n int) []model.Bar {
	bars := make([]model.Bar, n)
	for i := range bars {
		bars[i] = model.Bar{
			Symbol:    symbol,
			Close:     float64(100 + i),
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour).UTC(),
		}
	}
	return bars
}

// --- GetQuote tests ---

func TestMarketDataService_GetQuote(t *testing.T) {
	t.Parallel()

	t.Run("cache miss calls provider and returns quote", func(t *testing.T) {
		t.Parallel()
		calls := 0
		mock := &provider.MockProvider{
			GetQuoteFn: func(_ context.Context, symbol string) (*model.Quote, error) {
				calls++
				return fixedQuote(symbol, 150.0), nil
			},
		}
		svc := NewMarketDataService(mock)

		q, err := svc.GetQuote(context.Background(), "AAPL")
		if err != nil {
			t.Fatal(err)
		}
		if q.Price != 150.0 {
			t.Errorf("Price = %v, want 150.0", q.Price)
		}
		if calls != 1 {
			t.Errorf("provider calls = %d, want 1", calls)
		}
	})

	t.Run("cache hit skips provider on second call", func(t *testing.T) {
		t.Parallel()
		calls := 0
		mock := &provider.MockProvider{
			GetQuoteFn: func(_ context.Context, symbol string) (*model.Quote, error) {
				calls++
				return fixedQuote(symbol, 150.0), nil
			},
		}
		svc := NewMarketDataService(mock)

		_, _ = svc.GetQuote(context.Background(), "MSFT")
		_, _ = svc.GetQuote(context.Background(), "MSFT")

		if calls != 1 {
			t.Errorf("provider calls = %d, want 1 (second call should be cached)", calls)
		}
	})

	t.Run("expired cache entry re-fetches from provider", func(t *testing.T) {
		t.Parallel()
		calls := 0
		mock := &provider.MockProvider{
			GetQuoteFn: func(_ context.Context, symbol string) (*model.Quote, error) {
				calls++
				return fixedQuote(symbol, 150.0), nil
			},
		}
		svc := NewMarketDataService(mock)

		// Manually insert an expired entry.
		impl := svc.(*marketDataService)
		impl.mu.Lock()
		impl.quoteCache["TSLA"] = cachedQuote{
			quote:     fixedQuote("TSLA", 200.0),
			expiresAt: time.Now().Add(-1 * time.Second), // already expired
		}
		impl.mu.Unlock()

		q, err := svc.GetQuote(context.Background(), "TSLA")
		if err != nil {
			t.Fatal(err)
		}
		if q.Price != 150.0 {
			t.Errorf("Price = %v, want 150.0 (fresh from provider)", q.Price)
		}
		if calls != 1 {
			t.Errorf("provider calls = %d, want 1 (expired entry should re-fetch)", calls)
		}
	})

	t.Run("different symbols cached independently", func(t *testing.T) {
		t.Parallel()
		prices := map[string]float64{"AAPL": 150.0, "GOOG": 2800.0}
		mock := &provider.MockProvider{
			GetQuoteFn: func(_ context.Context, symbol string) (*model.Quote, error) {
				return fixedQuote(symbol, prices[symbol]), nil
			},
		}
		svc := NewMarketDataService(mock)

		qA, _ := svc.GetQuote(context.Background(), "AAPL")
		qG, _ := svc.GetQuote(context.Background(), "GOOG")

		if qA.Price != 150.0 {
			t.Errorf("AAPL price = %v, want 150.0", qA.Price)
		}
		if qG.Price != 2800.0 {
			t.Errorf("GOOG price = %v, want 2800.0", qG.Price)
		}
	})

	t.Run("provider error returns AppError with 502", func(t *testing.T) {
		t.Parallel()
		mock := &provider.MockProvider{
			GetQuoteFn: func(_ context.Context, _ string) (*model.Quote, error) {
				return nil, provider.ErrProviderUnavailable
			},
		}
		svc := NewMarketDataService(mock)

		_, err := svc.GetQuote(context.Background(), "AAPL")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var appErr *model.AppError
		if !errors.As(err, &appErr) {
			t.Errorf("error type = %T, want *model.AppError", err)
		}
		if appErr.HTTPStatus != 502 {
			t.Errorf("HTTPStatus = %d, want 502", appErr.HTTPStatus)
		}
	})

	t.Run("rate limited returns AppError with 429", func(t *testing.T) {
		t.Parallel()
		mock := &provider.MockProvider{
			GetQuoteFn: func(_ context.Context, _ string) (*model.Quote, error) {
				return nil, provider.ErrRateLimited
			},
		}
		svc := NewMarketDataService(mock)

		_, err := svc.GetQuote(context.Background(), "AAPL")
		var appErr *model.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
		if appErr.HTTPStatus != 429 {
			t.Errorf("HTTPStatus = %d, want 429", appErr.HTTPStatus)
		}
	})

	t.Run("invalid symbol returns AppError with 422", func(t *testing.T) {
		t.Parallel()
		mock := &provider.MockProvider{
			GetQuoteFn: func(_ context.Context, _ string) (*model.Quote, error) {
				return nil, provider.ErrInvalidSymbol
			},
		}
		svc := NewMarketDataService(mock)

		_, err := svc.GetQuote(context.Background(), "INVALID")
		var appErr *model.AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T", err)
		}
		if appErr.HTTPStatus != 422 {
			t.Errorf("HTTPStatus = %d, want 422", appErr.HTTPStatus)
		}
	})
}

// --- GetHistoricalBars tests ---

func TestMarketDataService_GetHistoricalBars(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	t.Run("cache miss calls provider and returns bars", func(t *testing.T) {
		t.Parallel()
		calls := 0
		mock := &provider.MockProvider{
			GetHistoricalBarsFn: func(_ context.Context, symbol string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
				calls++
				return fixedBars(symbol, 5), nil
			},
		}
		svc := NewMarketDataService(mock)

		bars, err := svc.GetHistoricalBars(context.Background(), "AAPL", model.Timeframe1M, start, end)
		if err != nil {
			t.Fatal(err)
		}
		if len(bars) != 5 {
			t.Errorf("len(bars) = %d, want 5", len(bars))
		}
		if calls != 1 {
			t.Errorf("provider calls = %d, want 1", calls)
		}
	})

	t.Run("cache hit skips provider on second identical request", func(t *testing.T) {
		t.Parallel()
		calls := 0
		mock := &provider.MockProvider{
			GetHistoricalBarsFn: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
				calls++
				return fixedBars("AAPL", 3), nil
			},
		}
		svc := NewMarketDataService(mock)

		_, _ = svc.GetHistoricalBars(context.Background(), "AAPL", model.Timeframe1M, start, end)
		_, _ = svc.GetHistoricalBars(context.Background(), "AAPL", model.Timeframe1M, start, end)

		if calls != 1 {
			t.Errorf("provider calls = %d, want 1 (second call should be cached)", calls)
		}
	})

	t.Run("provider error returns AppError", func(t *testing.T) {
		t.Parallel()
		mock := &provider.MockProvider{
			GetHistoricalBarsFn: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
				return nil, provider.ErrProviderUnavailable
			},
		}
		svc := NewMarketDataService(mock)

		_, err := svc.GetHistoricalBars(context.Background(), "AAPL", model.Timeframe1M, start, end)
		var appErr *model.AppError
		if !errors.As(err, &appErr) {
			t.Errorf("expected AppError, got %T: %v", err, err)
		}
	})
}

// --- Concurrency test ---

func TestMarketDataService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	calls := 0
	mock := &provider.MockProvider{
		GetQuoteFn: func(_ context.Context, symbol string) (*model.Quote, error) {
			mu.Lock()
			calls++
			mu.Unlock()
			return fixedQuote(symbol, 100.0), nil
		},
	}
	svc := NewMarketDataService(mock)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.GetQuote(context.Background(), "AAPL")
		}()
	}
	wg.Wait()

	// All 50 concurrent calls should result in very few provider calls
	// (ideally 1, but allow a small race window).
	if calls > 5 {
		t.Errorf("provider calls = %d under 50 concurrent requests; cache may not be working", calls)
	}
}
