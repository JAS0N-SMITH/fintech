package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// MockMarketDataProvider is a test double for MarketDataProvider.
type MockMarketDataProvider struct {
	GetQuoteFunc          func(ctx context.Context, symbol string) (*model.Quote, error)
	GetHistoricalBarsFunc func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error)
	StreamPricesFunc      func(ctx context.Context, symbols []string, handler func(model.PriceTick)) error
	GetSymbolsFunc        func(ctx context.Context, exchange string) ([]model.Symbol, error)
	HealthCheckFunc       func(ctx context.Context) error
}

func (m *MockMarketDataProvider) GetQuote(ctx context.Context, symbol string) (*model.Quote, error) {
	if m.GetQuoteFunc != nil {
		return m.GetQuoteFunc(ctx, symbol)
	}
	return nil, errors.New("not implemented")
}

func (m *MockMarketDataProvider) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	if m.GetHistoricalBarsFunc != nil {
		return m.GetHistoricalBarsFunc(ctx, symbol, tf, start, end)
	}
	return nil, errors.New("not implemented")
}

func (m *MockMarketDataProvider) StreamPrices(ctx context.Context, symbols []string, handler func(model.PriceTick)) error {
	if m.StreamPricesFunc != nil {
		return m.StreamPricesFunc(ctx, symbols, handler)
	}
	return errors.New("not implemented")
}

func (m *MockMarketDataProvider) GetSymbols(ctx context.Context, exchange string) ([]model.Symbol, error) {
	if m.GetSymbolsFunc != nil {
		return m.GetSymbolsFunc(ctx, exchange)
	}
	return nil, errors.New("not implemented")
}

func (m *MockMarketDataProvider) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return errors.New("not implemented")
}

func TestFallbackProvider_GetHistoricalBars_PrimarySucceeds(t *testing.T) {
	ctx := context.Background()
	bars := []model.Bar{{Symbol: "AAPL", Close: 150.0}}

	primary := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			return bars, nil
		},
	}

	fallback := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			t.Fatal("fallback should not be called when primary succeeds")
			return nil, nil
		},
	}

	fb := NewFallbackProvider(primary, fallback)
	result, err := fb.GetHistoricalBars(ctx, "AAPL", model.Timeframe1M, time.Now(), time.Now())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(result) != 1 || result[0].Close != 150.0 {
		t.Errorf("expected bars from primary, got %v", result)
	}
}

func TestFallbackProvider_GetHistoricalBars_PrimaryRateLimited(t *testing.T) {
	ctx := context.Background()

	primary := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			return nil, ErrRateLimited
		},
	}

	fallback := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			t.Fatal("fallback should NOT be called when primary is rate limited")
			return nil, nil
		},
	}

	fb := NewFallbackProvider(primary, fallback)
	_, err := fb.GetHistoricalBars(ctx, "AAPL", model.Timeframe1M, time.Now(), time.Now())

	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestFallbackProvider_GetHistoricalBars_PrimaryFailsFallbackSucceeds(t *testing.T) {
	ctx := context.Background()
	fallbackBars := []model.Bar{{Symbol: "AAPL", Close: 155.0}}

	primary := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			return nil, ErrProviderUnavailable
		},
	}

	fallback := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			return fallbackBars, nil
		},
	}

	fb := NewFallbackProvider(primary, fallback)
	result, err := fb.GetHistoricalBars(ctx, "AAPL", model.Timeframe1M, time.Now(), time.Now())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(result) != 1 || result[0].Close != 155.0 {
		t.Errorf("expected bars from fallback, got %v", result)
	}
}

func TestFallbackProvider_GetHistoricalBars_BothFail(t *testing.T) {
	ctx := context.Background()

	primary := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			return nil, ErrProviderUnavailable
		},
	}

	fallback := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			return nil, ErrInvalidSymbol
		},
	}

	fb := NewFallbackProvider(primary, fallback)
	_, err := fb.GetHistoricalBars(ctx, "AAPL", model.Timeframe1M, time.Now(), time.Now())

	// Should return fallback error
	if !errors.Is(err, ErrInvalidSymbol) {
		t.Errorf("expected ErrInvalidSymbol (from fallback), got %v", err)
	}
}

func TestFallbackProvider_GetHistoricalBars_NoFallback(t *testing.T) {
	ctx := context.Background()

	primary := &MockMarketDataProvider{
		GetHistoricalBarsFunc: func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
			return nil, ErrProviderUnavailable
		},
	}

	fb := NewFallbackProvider(primary, nil)
	_, err := fb.GetHistoricalBars(ctx, "AAPL", model.Timeframe1M, time.Now(), time.Now())

	// Should return primary error when no fallback
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable (from primary), got %v", err)
	}
}

func TestFallbackProvider_DelegatesOtherMethods(t *testing.T) {
	ctx := context.Background()

	primary := &MockMarketDataProvider{
		GetQuoteFunc: func(ctx context.Context, symbol string) (*model.Quote, error) {
			return &model.Quote{Symbol: "AAPL"}, nil
		},
		GetSymbolsFunc: func(ctx context.Context, exchange string) ([]model.Symbol, error) {
			return []model.Symbol{{Symbol: "AAPL"}}, nil
		},
		HealthCheckFunc: func(ctx context.Context) error {
			return nil
		},
	}

	fallback := &MockMarketDataProvider{} // Should not be called

	fb := NewFallbackProvider(primary, fallback)

	// GetQuote should delegate to primary
	quote, err := fb.GetQuote(ctx, "AAPL")
	if err != nil || quote.Symbol != "AAPL" {
		t.Errorf("GetQuote failed: %v", err)
	}

	// GetSymbols should delegate to primary
	symbols, err := fb.GetSymbols(ctx, "US")
	if err != nil || len(symbols) == 0 {
		t.Errorf("GetSymbols failed: %v", err)
	}

	// HealthCheck should delegate to primary
	err = fb.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}
