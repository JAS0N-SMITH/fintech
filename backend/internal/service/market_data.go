package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/huchknows/fintech/backend/internal/config"
	"github.com/huchknows/fintech/backend/internal/model"
	"github.com/huchknows/fintech/backend/internal/provider"
)

// quoteTTL is how long a real-time quote remains valid in the cache.
const quoteTTL = time.Duration(config.QuoteCacheTTL) * time.Second

// historicalTTL is how long historical bar data is cached.
const historicalTTL = time.Duration(config.HistoricalCacheTTL) * time.Second

// cachedQuote holds a Quote with its expiry time.
type cachedQuote struct {
	quote     *model.Quote
	expiresAt time.Time
}

// cachedBars holds bar data with its expiry time.
// The cache key encodes symbol + timeframe + date range to avoid stale matches.
type cachedBars struct {
	bars      []model.Bar
	expiresAt time.Time
}

// MarketDataService fetches and caches market data from a MarketDataProvider.
// All methods are safe for concurrent use.
type MarketDataService interface {
	// GetQuote returns a fresh or cached full quote for the given symbol.
	GetQuote(ctx context.Context, symbol string) (*model.Quote, error)

	// GetHistoricalBars returns OHLCV candle data for the given symbol and range.
	GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error)
}

type marketDataService struct {
	provider   provider.MarketDataProvider
	quoteCache map[string]cachedQuote
	barCache   map[string]cachedBars
	mu         sync.RWMutex
}

// NewMarketDataService creates a MarketDataService backed by the given provider.
func NewMarketDataService(p provider.MarketDataProvider) MarketDataService {
	return &marketDataService{
		provider:   p,
		quoteCache: make(map[string]cachedQuote),
		barCache:   make(map[string]cachedBars),
	}
}

// GetQuote returns a cached quote if still fresh, otherwise fetches from the provider.
func (s *marketDataService) GetQuote(ctx context.Context, symbol string) (*model.Quote, error) {
	// Fast path: read lock, check cache.
	s.mu.RLock()
	if entry, ok := s.quoteCache[symbol]; ok && time.Now().Before(entry.expiresAt) {
		s.mu.RUnlock()
		return entry.quote, nil
	}
	s.mu.RUnlock()

	// Cache miss or expired: fetch from provider.
	quote, err := s.provider.GetQuote(ctx, symbol)
	if err != nil {
		return nil, mapProviderError(err, symbol)
	}

	// Write lock: store in cache.
	s.mu.Lock()
	s.quoteCache[symbol] = cachedQuote{
		quote:     quote,
		expiresAt: time.Now().Add(quoteTTL),
	}
	s.mu.Unlock()

	return quote, nil
}

// GetHistoricalBars returns cached bars if still fresh, otherwise fetches from the provider.
// The cache key encodes all request parameters to avoid cross-contamination.
func (s *marketDataService) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	key := fmt.Sprintf("%s:%s:%d:%d", symbol, tf, start.Unix(), end.Unix())

	// Fast path: read lock, check cache.
	s.mu.RLock()
	if entry, ok := s.barCache[key]; ok && time.Now().Before(entry.expiresAt) {
		s.mu.RUnlock()
		return entry.bars, nil
	}
	s.mu.RUnlock()

	// Cache miss or expired: fetch from provider.
	bars, err := s.provider.GetHistoricalBars(ctx, symbol, tf, start, end)
	if err != nil {
		return nil, mapProviderError(err, symbol)
	}

	// Write lock: store in cache.
	s.mu.Lock()
	s.barCache[key] = cachedBars{
		bars:      bars,
		expiresAt: time.Now().Add(historicalTTL),
	}
	s.mu.Unlock()

	return bars, nil
}

// mapProviderError translates provider sentinel errors into AppErrors with
// appropriate HTTP status codes. Internal details are never surfaced to callers.
func mapProviderError(err error, symbol string) *model.AppError {
	switch {
	case errors.Is(err, provider.ErrInvalidSymbol):
		return &model.AppError{
			Code:       model.ErrValidation,
			Message:    fmt.Sprintf("symbol %q is not recognised", symbol),
			HTTPStatus: http.StatusUnprocessableEntity, // 422
		}
	case errors.Is(err, provider.ErrRateLimited):
		return &model.AppError{
			Code:       model.ErrConflict,
			Message:    "market data service is temporarily rate limited, please retry",
			HTTPStatus: http.StatusTooManyRequests, // 429
		}
	default:
		return &model.AppError{
			Code:       errors.New("provider error"),
			Message:    "market data is temporarily unavailable",
			HTTPStatus: http.StatusBadGateway, // 502
		}
	}
}
