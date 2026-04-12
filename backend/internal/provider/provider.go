// Package provider defines the MarketDataProvider interface and concrete
// implementations for fetching market data from external sources.
//
// The interface decouples the service layer from any specific vendor,
// making providers swappable (e.g. Finnhub → Polygon) without changing
// business logic. See ADR 004 for the design rationale.
package provider

import (
	"context"
	"errors"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// Sentinel errors returned by all MarketDataProvider implementations.
// Callers use errors.Is() to inspect these without depending on provider internals.
var (
	// ErrInvalidSymbol is returned when the requested ticker symbol is not
	// recognised by the data provider.
	ErrInvalidSymbol = errors.New("invalid symbol")

	// ErrRateLimited is returned when the provider's request quota is exceeded.
	ErrRateLimited = errors.New("rate limited")

	// ErrProviderUnavailable is returned when the provider cannot be reached
	// or returns an unexpected server-side error.
	ErrProviderUnavailable = errors.New("provider unavailable")
)

// MarketDataProvider is the interface that all market data sources must satisfy.
// Implementations must be safe for concurrent use.
type MarketDataProvider interface {
	// GetQuote returns a full market snapshot for the given symbol.
	// Used for the initial REST fetch when a component mounts.
	GetQuote(ctx context.Context, symbol string) (*model.Quote, error)

	// GetHistoricalBars returns OHLCV candles for the given symbol and timeframe.
	// start and end are inclusive; timeframe controls candle resolution.
	GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error)

	// StreamPrices opens a real-time price stream for the given symbols and
	// calls handler for each incoming PriceTick. The stream runs until ctx
	// is cancelled or a non-recoverable error occurs.
	// Implementations must call handler from a single goroutine (no concurrent calls).
	StreamPrices(ctx context.Context, symbols []string, handler func(model.PriceTick)) error

	// HealthCheck verifies that the provider is operational.
	// Uses cached status with a short TTL to avoid exhausting rate limits.
	HealthCheck(ctx context.Context) error
}
