package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// mockProvider is a test double for MarketDataProvider.
type mockProvider struct {
	getHistoricalBarsFunc func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error)
	getQuoteFunc          func(ctx context.Context, symbol string) (*model.Quote, error)
	getSymbolsFunc        func(ctx context.Context, exchange string) ([]model.Symbol, error)
	healthCheckFunc       func(ctx context.Context) error
}

func (m *mockProvider) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	if m.getHistoricalBarsFunc != nil {
		return m.getHistoricalBarsFunc(ctx, symbol, tf, start, end)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProvider) GetQuote(ctx context.Context, symbol string) (*model.Quote, error) {
	if m.getQuoteFunc != nil {
		return m.getQuoteFunc(ctx, symbol)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProvider) StreamPrices(_ context.Context, _ []string, _ func(model.PriceTick)) error {
	return errors.New("not implemented")
}

func (m *mockProvider) GetSymbols(ctx context.Context, exchange string) ([]model.Symbol, error) {
	if m.getSymbolsFunc != nil {
		return m.getSymbolsFunc(ctx, exchange)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProvider) HealthCheck(ctx context.Context) error {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return errors.New("not implemented")
}

// --- GetHistoricalBars routing tests ---

func TestFallbackProvider_GetHistoricalBars_PolygonSucceeds(t *testing.T) {
	ctx := context.Background()
	polygonBars := []model.Bar{{Symbol: "AAPL", Close: 175.0}}

	polygon := &mockProvider{
		getHistoricalBarsFunc: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
			return polygonBars, nil
		},
	}
	finnhub := &mockProvider{
		getHistoricalBarsFunc: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
			t.Fatal("finnhub should not be called when polygon succeeds")
			return nil, nil
		},
	}

	fp := NewFallbackProvider(finnhub, polygon)
	bars, err := fp.GetHistoricalBars(ctx, "AAPL", model.Timeframe1M, time.Now(), time.Now())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bars) != 1 || bars[0].Close != 175.0 {
		t.Errorf("expected polygon bars, got %v", bars)
	}
}

func TestFallbackProvider_GetHistoricalBars_PolygonFails_ErrorPropagates(t *testing.T) {
	ctx := context.Background()

	polygon := &mockProvider{
		getHistoricalBarsFunc: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
			return nil, ErrProviderUnavailable
		},
	}
	finnhub := &mockProvider{
		getHistoricalBarsFunc: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
			t.Fatal("finnhub should not be called for historical bars")
			return nil, nil
		},
	}

	fp := NewFallbackProvider(finnhub, polygon)
	_, err := fp.GetHistoricalBars(ctx, "AAPL", model.Timeframe1M, time.Now(), time.Now())

	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable from polygon, got %v", err)
	}
}

func TestFallbackProvider_GetHistoricalBars_PolygonRateLimited_ErrorPropagates(t *testing.T) {
	ctx := context.Background()

	polygon := &mockProvider{
		getHistoricalBarsFunc: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
			return nil, ErrRateLimited
		},
	}
	finnhub := &mockProvider{
		getHistoricalBarsFunc: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
			t.Fatal("finnhub should not be called for historical bars")
			return nil, nil
		},
	}

	fp := NewFallbackProvider(finnhub, polygon)
	_, err := fp.GetHistoricalBars(ctx, "TSLA", model.Timeframe1M, time.Now(), time.Now())

	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited from polygon, got %v", err)
	}
}

func TestFallbackProvider_GetHistoricalBars_NoPolygon_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()

	finnhub := &mockProvider{
		getHistoricalBarsFunc: func(_ context.Context, _ string, _ model.Timeframe, _, _ time.Time) ([]model.Bar, error) {
			t.Fatal("finnhub should not be called for historical bars")
			return nil, nil
		},
	}

	fp := NewFallbackProvider(finnhub, nil)
	bars, err := fp.GetHistoricalBars(ctx, "MSFT", model.Timeframe1M, time.Now(), time.Now())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bars) != 0 {
		t.Errorf("expected empty bars when no polygon configured, got %v", bars)
	}
}

// --- Non-bars methods always route to Finnhub ---

func TestFallbackProvider_RealtimeMethods_AlwaysRoutedToFinnhub(t *testing.T) {
	ctx := context.Background()

	finnhub := &mockProvider{
		getQuoteFunc: func(_ context.Context, symbol string) (*model.Quote, error) {
			return &model.Quote{Symbol: symbol, Price: 150.0}, nil
		},
		getSymbolsFunc: func(_ context.Context, _ string) ([]model.Symbol, error) {
			return []model.Symbol{{Symbol: "AAPL"}}, nil
		},
		healthCheckFunc: func(_ context.Context) error { return nil },
	}
	polygon := &mockProvider{
		getQuoteFunc: func(_ context.Context, _ string) (*model.Quote, error) {
			t.Fatal("polygon should never be called for GetQuote")
			return nil, nil
		},
		getSymbolsFunc: func(_ context.Context, _ string) ([]model.Symbol, error) {
			t.Fatal("polygon should never be called for GetSymbols")
			return nil, nil
		},
	}

	fp := NewFallbackProvider(finnhub, polygon)

	quote, err := fp.GetQuote(ctx, "AAPL")
	if err != nil || quote.Price != 150.0 {
		t.Errorf("GetQuote: expected finnhub result, got err=%v quote=%v", err, quote)
	}

	syms, err := fp.GetSymbols(ctx, "US")
	if err != nil || len(syms) == 0 {
		t.Errorf("GetSymbols: expected finnhub result, got err=%v", err)
	}

	if err := fp.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck: unexpected error: %v", err)
	}
}
