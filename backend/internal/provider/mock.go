package provider

import (
	"context"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// MockProvider is a configurable test double for MarketDataProvider.
// Set the Fn fields to control what each method returns.
type MockProvider struct {
	GetQuoteFn          func(ctx context.Context, symbol string) (*model.Quote, error)
	GetHistoricalBarsFn func(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error)
	StreamPricesFn      func(ctx context.Context, symbols []string, handler func(model.PriceTick)) error
	GetSymbolsFn        func(ctx context.Context, exchange string) ([]model.Symbol, error)
	HealthCheckFn       func(ctx context.Context) error
}

// GetQuote delegates to GetQuoteFn if set, otherwise returns a zero Quote.
func (m *MockProvider) GetQuote(ctx context.Context, symbol string) (*model.Quote, error) {
	if m.GetQuoteFn != nil {
		return m.GetQuoteFn(ctx, symbol)
	}
	return &model.Quote{Symbol: symbol}, nil
}

// GetHistoricalBars delegates to GetHistoricalBarsFn if set, otherwise returns an empty slice.
func (m *MockProvider) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	if m.GetHistoricalBarsFn != nil {
		return m.GetHistoricalBarsFn(ctx, symbol, tf, start, end)
	}
	return []model.Bar{}, nil
}

// StreamPrices delegates to StreamPricesFn if set, otherwise returns immediately.
func (m *MockProvider) StreamPrices(ctx context.Context, symbols []string, handler func(model.PriceTick)) error {
	if m.StreamPricesFn != nil {
		return m.StreamPricesFn(ctx, symbols, handler)
	}
	<-ctx.Done()
	return nil
}

// GetSymbols delegates to GetSymbolsFn if set, otherwise returns an empty slice.
func (m *MockProvider) GetSymbols(ctx context.Context, exchange string) ([]model.Symbol, error) {
	if m.GetSymbolsFn != nil {
		return m.GetSymbolsFn(ctx, exchange)
	}
	return []model.Symbol{}, nil
}

// HealthCheck delegates to HealthCheckFn if set, otherwise returns nil.
func (m *MockProvider) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFn != nil {
		return m.HealthCheckFn(ctx)
	}
	return nil
}

// Compile-time interface compliance check.
var _ MarketDataProvider = (*MockProvider)(nil)
