package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/huchknows/fintech/backend/internal/model"
)

// PolygonProvider implements partial MarketDataProvider using the Polygon.io API.
// Polygon provides historical bar data but not real-time streaming on the free tier.
// REST docs: https://polygon.io/docs/stocks/getting-started
// Free tier: 5 API calls/min
type PolygonProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewPolygonProvider creates a PolygonProvider with the given API key.
func NewPolygonProvider(apiKey string) *PolygonProvider {
	return &PolygonProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// polygonAgg represents a single aggregate (candle) from Polygon API.
type polygonAgg struct {
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Close     float64 `json:"c"`
	Volume    float64 `json:"v"`
	Timestamp int64   `json:"t"` // Unix milliseconds
}

// polygonBarsResponse maps the Polygon /aggs/ticker/{ticker}/range response.
type polygonBarsResponse struct {
	Results []polygonAgg `json:"results"`
	Status  string       `json:"status"`
	Count   int          `json:"count"`
}

// GetHistoricalBars fetches OHLCV candles for the given symbol and timeframe from Polygon.
// Converts app timeframe to Polygon multiplier and timespan.
func (p *PolygonProvider) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	// Map app timeframes to Polygon timespan/multiplier
	// Polygon timespan: minute, hour, day, week, month, quarter, year
	// multiplier controls the granularity (e.g., 5 minute, 15 minute, etc)
	var timespan string
	var multiplier int

	switch tf {
	case model.Timeframe1D:
		// 5-minute candles for intraday
		timespan = "minute"
		multiplier = 5
	case model.Timeframe1W:
		// Daily candles for weekly view
		timespan = "day"
		multiplier = 1
	case model.Timeframe1M:
		// Daily candles for monthly view
		timespan = "day"
		multiplier = 1
	case model.Timeframe3M:
		// Daily candles for quarterly view
		timespan = "day"
		multiplier = 1
	case model.Timeframe1Y:
		// Weekly candles for yearly view
		timespan = "week"
		multiplier = 1
	case model.TimeframeAll:
		// Monthly candles for all time
		timespan = "month"
		multiplier = 1
	default:
		return nil, fmt.Errorf("%w: unknown timeframe %q", ErrInvalidSymbol, tf)
	}

	// Polygon requires dates in YYYY-MM-DD format
	fromStr := start.Format("2006-01-02")
	toStr := end.Format("2006-01-02")

	endpoint := fmt.Sprintf(
		"https://api.polygon.io/v2/aggs/ticker/%s/range/%d/%s/%s/%s?apikey=%s&sort=asc&limit=50000",
		url.QueryEscape(symbol),
		multiplier,
		timespan,
		fromStr,
		toStr,
		p.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnavailable, err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnavailable, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		// handled below
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case http.StatusNotFound, http.StatusUnprocessableEntity:
		// Symbol not found or unsupported
		return nil, ErrInvalidSymbol
	case http.StatusUnauthorized, http.StatusForbidden:
		// Invalid API key or insufficient permissions
		return nil, ErrProviderUnavailable
	default:
		return nil, fmt.Errorf("%w: HTTP %d", ErrProviderUnavailable, resp.StatusCode)
	}

	var raw polygonBarsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: decode bars: %s", ErrProviderUnavailable, err)
	}

	if raw.Status != "OK" || len(raw.Results) == 0 {
		return []model.Bar{}, nil
	}

	bars := make([]model.Bar, len(raw.Results))
	for i, agg := range raw.Results {
		// Polygon returns timestamps in milliseconds; convert to seconds
		ts := time.Unix(0, agg.Timestamp*int64(time.Millisecond)).UTC()
		bars[i] = model.Bar{
			Symbol:    symbol,
			Open:      agg.Open,
			High:      agg.High,
			Low:       agg.Low,
			Close:     agg.Close,
			Volume:    int64(agg.Volume),
			Timestamp: ts,
		}
	}

	return bars, nil
}

// GetQuote is not implemented for Polygon (use Finnhub instead).
func (p *PolygonProvider) GetQuote(_ context.Context, _ string) (*model.Quote, error) {
	return nil, fmt.Errorf("%w: GetQuote not implemented for Polygon provider", ErrProviderUnavailable)
}

// StreamPrices is not implemented for Polygon (free tier does not support streaming).
func (p *PolygonProvider) StreamPrices(_ context.Context, _ []string, _ func(model.PriceTick)) error {
	return fmt.Errorf("%w: StreamPrices not implemented for Polygon provider", ErrProviderUnavailable)
}

// GetSymbols is not implemented for Polygon (use Finnhub instead).
func (p *PolygonProvider) GetSymbols(_ context.Context, _ string) ([]model.Symbol, error) {
	return nil, fmt.Errorf("%w: GetSymbols not implemented for Polygon provider", ErrProviderUnavailable)
}

// HealthCheck is not implemented for Polygon.
func (p *PolygonProvider) HealthCheck(_ context.Context) error {
	return fmt.Errorf("%w: HealthCheck not implemented for Polygon provider", ErrProviderUnavailable)
}
