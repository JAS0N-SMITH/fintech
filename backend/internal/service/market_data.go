package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// cachedSymbols holds the exchange symbol list with its expiry time.
type cachedSymbols struct {
	symbols   []model.Symbol
	expiresAt time.Time
}

// MarketDataService fetches and caches market data from a MarketDataProvider.
// All methods are safe for concurrent use.
type MarketDataService interface {
	// GetQuote returns a fresh or cached full quote for the given symbol.
	GetQuote(ctx context.Context, symbol string) (*model.Quote, error)

	// GetHistoricalBars returns OHLCV candle data for the given symbol and range.
	GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error)

	// SearchSymbols returns a list of supported symbols matching the query.
	// query filters by case-insensitive prefix match on symbol and substring match on description.
	// limit caps the number of results returned (1-50); results are truncated to this count.
	SearchSymbols(ctx context.Context, query string, limit int) ([]model.Symbol, error)
}

type marketDataService struct {
	provider    provider.MarketDataProvider
	quoteCache  map[string]cachedQuote
	barCache    map[string]cachedBars
	symbolCache *cachedSymbols
	mu          sync.RWMutex
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
// The cache key normalises start/end to UTC calendar dates so that repeated page loads
// with slightly different sub-second timestamps share the same cache entry.
func (s *marketDataService) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	startDay := start.UTC().Truncate(24 * time.Hour)
	endDay := end.UTC().Truncate(24 * time.Hour)
	key := fmt.Sprintf("%s:%s:%s:%s", symbol, tf, startDay.Format("2006-01-02"), endDay.Format("2006-01-02"))

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

	// Only cache non-empty results — empty means the provider had no data, and
	// a subsequent request (e.g., after the market closes) may succeed.
	if len(bars) > 0 {
		s.mu.Lock()
		s.barCache[key] = cachedBars{
			bars:      bars,
			expiresAt: time.Now().Add(historicalTTL),
		}
		s.mu.Unlock()
	}

	return bars, nil
}

// SearchSymbols returns a filtered list of supported symbols for the US exchange.
// Filters by case-insensitive prefix match on symbol and substring match on description.
// Results are capped at limit (1-50); if limit is invalid, defaults to 20.
func (s *marketDataService) SearchSymbols(ctx context.Context, query string, limit int) ([]model.Symbol, error) {
	// Normalize limit
	if limit < 1 || limit > 50 {
		limit = 20
	}

	// Fast path: read lock, check cache.
	s.mu.RLock()
	if s.symbolCache != nil && time.Now().Before(s.symbolCache.expiresAt) {
		symbols := s.symbolCache.symbols
		s.mu.RUnlock()
		return filterSymbols(symbols, query, limit), nil
	}
	s.mu.RUnlock()

	// Cache miss or expired: fetch from provider.
	symbols, err := s.provider.GetSymbols(ctx, "US")
	if err != nil {
		return nil, mapProviderError(err, "")
	}

	// Write lock: store in cache.
	s.mu.Lock()
	s.symbolCache = &cachedSymbols{
		symbols:   symbols,
		expiresAt: time.Now().Add(time.Duration(config.SymbolsCacheTTL) * time.Second),
	}
	s.mu.Unlock()

	return filterSymbols(symbols, query, limit), nil
}

// filterSymbols returns symbols matching the query (case-insensitive prefix on symbol,
// substring on description), limited to the given count.
func filterSymbols(symbols []model.Symbol, query string, limit int) []model.Symbol {
	if query == "" {
		// No query: return first N symbols
		if len(symbols) > limit {
			return symbols[:limit]
		}
		return symbols
	}

	query = strings.ToUpper(query)
	var results []model.Symbol
	for _, sym := range symbols {
		// Case-insensitive prefix match on symbol
		if strings.HasPrefix(strings.ToUpper(sym.Symbol), query) {
			results = append(results, sym)
			if len(results) >= limit {
				break
			}
			continue
		}
		// Case-insensitive substring match on description
		if strings.Contains(strings.ToUpper(sym.Description), query) {
			results = append(results, sym)
			if len(results) >= limit {
				break
			}
		}
	}
	return results
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
