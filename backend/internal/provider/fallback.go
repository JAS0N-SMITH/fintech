package provider

import (
	"context"
	"log/slog"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// FallbackProvider routes market data to specialized providers per ADR 015:
//   - Real-time data (GetQuote, StreamPrices, GetSymbols, HealthCheck): always routes to realtime (Finnhub).
//   - Historical bars (GetHistoricalBars): routes exclusively to historical (Polygon); returns empty if Polygon is nil or has no data.
type FallbackProvider struct {
	realtime   MarketDataProvider // Finnhub: quotes, symbols, streaming
	historical MarketDataProvider // Polygon: historical bars (optional; may be nil)
}

// NewFallbackProvider creates a FallbackProvider.
// realtime is required (Finnhub). historical is optional (Polygon); pass nil to disable.
func NewFallbackProvider(realtime, historical MarketDataProvider) *FallbackProvider {
	return &FallbackProvider{
		realtime:   realtime,
		historical: historical,
	}
}

// GetQuote delegates to the realtime provider.
func (p *FallbackProvider) GetQuote(ctx context.Context, symbol string) (*model.Quote, error) {
	return p.realtime.GetQuote(ctx, symbol)
}

// GetHistoricalBars routes exclusively to the historical provider (Polygon).
// Returns empty bars if Polygon is not configured or returns an error.
func (p *FallbackProvider) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	if p.historical == nil {
		slog.Warn("no historical provider configured, returning empty bars", "symbol", symbol)
		return []model.Bar{}, nil
	}
	bars, err := p.historical.GetHistoricalBars(ctx, symbol, tf, start, end)
	if err != nil {
		slog.Error("polygon historical bars failed", "symbol", symbol, "error", err)
		return nil, err
	}
	slog.Debug("historical bars served by polygon", "symbol", symbol, "timeframe", tf)
	return bars, nil
}

// StreamPrices delegates to the realtime provider.
func (p *FallbackProvider) StreamPrices(ctx context.Context, symbols []string, handler func(model.PriceTick)) error {
	return p.realtime.StreamPrices(ctx, symbols, handler)
}

// GetSymbols delegates to the realtime provider.
func (p *FallbackProvider) GetSymbols(ctx context.Context, exchange string) ([]model.Symbol, error) {
	return p.realtime.GetSymbols(ctx, exchange)
}

// HealthCheck delegates to the realtime provider.
func (p *FallbackProvider) HealthCheck(ctx context.Context) error {
	return p.realtime.HealthCheck(ctx)
}
