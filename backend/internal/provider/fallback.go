package provider

import (
	"context"
	"errors"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// FallbackProvider wraps a primary provider with an optional fallback provider.
// For GetHistoricalBars, it tries the primary provider first; if that fails with
// ErrInvalidSymbol or ErrProviderUnavailable (but not ErrRateLimited), it tries
// the fallback provider if available.
//
// All other methods delegate to the primary provider.
type FallbackProvider struct {
	primary  MarketDataProvider
	fallback MarketDataProvider // optional; may be nil
}

// NewFallbackProvider creates a FallbackProvider with primary and optional fallback.
func NewFallbackProvider(primary, fallback MarketDataProvider) *FallbackProvider {
	return &FallbackProvider{
		primary:  primary,
		fallback: fallback,
	}
}

// GetQuote delegates to primary provider.
func (p *FallbackProvider) GetQuote(ctx context.Context, symbol string) (*model.Quote, error) {
	return p.primary.GetQuote(ctx, symbol)
}

// GetHistoricalBars tries primary first; if it fails, falls back to the fallback provider.
// If fallback is nil or also fails, returns the fallback error. If both are unavailable,
// returns the fallback error (last attempted provider's error).
func (p *FallbackProvider) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	// Try primary provider first
	bars, err := p.primary.GetHistoricalBars(ctx, symbol, tf, start, end)
	if err == nil {
		return bars, nil
	}

	// If primary succeeded (err == nil), we already returned above.
	// If primary failed with rate limit, don't try fallback.
	if errors.Is(err, ErrRateLimited) {
		return nil, err
	}

	// If fallback is available, try it
	if p.fallback != nil {
		fallbackBars, fallbackErr := p.fallback.GetHistoricalBars(ctx, symbol, tf, start, end)
		if fallbackErr == nil {
			return fallbackBars, nil
		}
		// Fallback failed; return fallback error
		return nil, fallbackErr
	}

	// No fallback available; return primary error
	return nil, err
}

// StreamPrices delegates to primary provider.
func (p *FallbackProvider) StreamPrices(ctx context.Context, symbols []string, handler func(model.PriceTick)) error {
	return p.primary.StreamPrices(ctx, symbols, handler)
}

// GetSymbols delegates to primary provider.
func (p *FallbackProvider) GetSymbols(ctx context.Context, exchange string) ([]model.Symbol, error) {
	return p.primary.GetSymbols(ctx, exchange)
}

// HealthCheck delegates to primary provider.
func (p *FallbackProvider) HealthCheck(ctx context.Context) error {
	return p.primary.HealthCheck(ctx)
}
