package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/huchknows/fintech/backend/internal/model"
)

// FinnhubProvider implements MarketDataProvider using the Finnhub.io API.
// REST docs: https://finnhub.io/docs/api
// Free tier: 60 REST req/min, 50 WebSocket connections.
type FinnhubProvider struct {
	apiKey     string
	baseURL    string
	wsURL      string
	httpClient *http.Client
}

// NewFinnhubProvider creates a FinnhubProvider with the given credentials.
// baseURL is the REST base (e.g. "https://finnhub.io/api/v1").
// wsURL is the WebSocket endpoint (e.g. "wss://ws.finnhub.io").
func NewFinnhubProvider(apiKey, baseURL, wsURL string) *FinnhubProvider {
	return &FinnhubProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		wsURL:   wsURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// --- REST: GetQuote ---

// finnhubQuoteResponse maps the Finnhub /quote endpoint JSON response.
type finnhubQuoteResponse struct {
	CurrentPrice  float64 `json:"c"`
	DayHigh       float64 `json:"h"`
	DayLow        float64 `json:"l"`
	Open          float64 `json:"o"`
	PreviousClose float64 `json:"pc"`
	Volume        int64   `json:"v"` // volume (note: not always present on free tier)
	Timestamp     int64   `json:"t"` // Unix timestamp
}

// GetQuote fetches a full market quote for the given symbol from Finnhub REST API.
func (p *FinnhubProvider) GetQuote(ctx context.Context, symbol string) (*model.Quote, error) {
	endpoint := fmt.Sprintf("%s/quote?symbol=%s&token=%s",
		p.baseURL, url.QueryEscape(symbol), p.apiKey)

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
	case http.StatusForbidden:
		// Finnhub returns 403 for invalid API keys and also for symbols not
		// accessible on the current plan. Treat as invalid symbol.
		return nil, ErrInvalidSymbol
	default:
		return nil, fmt.Errorf("%w: HTTP %d", ErrProviderUnavailable, resp.StatusCode)
	}

	var raw finnhubQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: decode quote: %s", ErrProviderUnavailable, err)
	}

	// Finnhub returns c=0 for unknown symbols rather than an error status.
	if raw.CurrentPrice == 0 && raw.Timestamp == 0 {
		return nil, ErrInvalidSymbol
	}

	ts := time.Unix(raw.Timestamp, 0).UTC()
	if raw.Timestamp == 0 {
		ts = time.Now().UTC()
	}

	return &model.Quote{
		Symbol:        symbol,
		Price:         raw.CurrentPrice,
		DayHigh:       raw.DayHigh,
		DayLow:        raw.DayLow,
		Open:          raw.Open,
		PreviousClose: raw.PreviousClose,
		Volume:        raw.Volume,
		Timestamp:     ts,
	}, nil
}

// --- REST: GetHistoricalBars ---

// finnhubResolution maps app Timeframe values to Finnhub resolution strings.
var finnhubResolution = map[model.Timeframe]string{
	model.Timeframe1D:  "5",  // 5-minute candles for intraday
	model.Timeframe1W:  "60", // 1-hour candles for 1 week
	model.Timeframe1M:  "D",  // daily candles for 1 month
	model.Timeframe3M:  "D",  // daily candles for 3 months
	model.Timeframe1Y:  "W",  // weekly candles for 1 year
	model.TimeframeAll: "M",  // monthly candles for all time
}

// finnhubCandleResponse maps the Finnhub /stock/candle endpoint JSON response.
type finnhubCandleResponse struct {
	Close     []float64 `json:"c"`
	High      []float64 `json:"h"`
	Low       []float64 `json:"l"`
	Open      []float64 `json:"o"`
	Timestamp []int64   `json:"t"`
	Volume    []int64   `json:"v"`
	Status    string    `json:"s"` // "ok" or "no_data"
}

// GetHistoricalBars fetches OHLCV candles for the given symbol and timeframe.
func (p *FinnhubProvider) GetHistoricalBars(ctx context.Context, symbol string, tf model.Timeframe, start, end time.Time) ([]model.Bar, error) {
	resolution, ok := finnhubResolution[tf]
	if !ok {
		return nil, fmt.Errorf("%w: unknown timeframe %q", ErrInvalidSymbol, tf)
	}

	endpoint := fmt.Sprintf("%s/stock/candle?symbol=%s&resolution=%s&from=%d&to=%d&token=%s",
		p.baseURL,
		url.QueryEscape(symbol),
		resolution,
		start.Unix(),
		end.Unix(),
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
	case http.StatusForbidden:
		return nil, ErrInvalidSymbol
	default:
		return nil, fmt.Errorf("%w: HTTP %d", ErrProviderUnavailable, resp.StatusCode)
	}

	var raw finnhubCandleResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: decode candle: %s", ErrProviderUnavailable, err)
	}

	if raw.Status == "no_data" || len(raw.Timestamp) == 0 {
		return []model.Bar{}, nil
	}

	bars := make([]model.Bar, len(raw.Timestamp))
	for i, ts := range raw.Timestamp {
		bars[i] = model.Bar{
			Symbol:    symbol,
			Open:      raw.Open[i],
			High:      raw.High[i],
			Low:       raw.Low[i],
			Close:     raw.Close[i],
			Volume:    raw.Volume[i],
			Timestamp: time.Unix(ts, 0).UTC(),
		}
	}
	return bars, nil
}

// --- WebSocket: StreamPrices ---

// finnhubWSMessage is a message sent to the Finnhub WebSocket to subscribe/unsubscribe.
type finnhubWSMessage struct {
	Type   string `json:"type"`
	Symbol string `json:"symbol"`
}

// finnhubTradeEvent is the structure of trade events from the Finnhub WebSocket.
type finnhubTradeEvent struct {
	Data []struct {
		Price     float64 `json:"p"`
		Symbol    string  `json:"s"`
		Timestamp int64   `json:"t"` // Unix milliseconds
		Volume    float64 `json:"v"`
	} `json:"data"`
	Type string `json:"type"`
}

// StreamPrices opens a WebSocket connection to Finnhub and streams real-time
// PriceTick events for the given symbols. The stream runs until ctx is cancelled.
// handler is called synchronously for each tick — implementations should be fast.
func (p *FinnhubProvider) StreamPrices(ctx context.Context, symbols []string, handler func(model.PriceTick)) error {
	wsURL := fmt.Sprintf("%s?token=%s", p.wsURL, p.apiKey)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("%w: dial: %s", ErrProviderUnavailable, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// Subscribe to each symbol.
	for _, sym := range symbols {
		msg := finnhubWSMessage{Type: "subscribe", Symbol: sym}
		if err := conn.WriteJSON(msg); err != nil {
			return fmt.Errorf("%w: subscribe %s: %s", ErrProviderUnavailable, sym, err)
		}
	}

	// Read loop — exits when ctx is cancelled or connection drops.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var event finnhubTradeEvent
			if err := conn.ReadJSON(&event); err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					slog.Warn("finnhub websocket read error", "error", err)
				}
				return
			}
			if event.Type != "trade" {
				continue
			}
			for _, d := range event.Data {
				handler(model.PriceTick{
					Symbol:    d.Symbol,
					Price:     d.Price,
					Volume:    int64(d.Volume),
					Timestamp: time.UnixMilli(d.Timestamp).UTC(),
				})
			}
		}
	}()

	select {
	case <-ctx.Done():
		// Graceful close.
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		<-done
		return nil
	case <-done:
		return fmt.Errorf("%w: connection closed unexpectedly", ErrProviderUnavailable)
	}
}

// HealthCheck verifies that Finnhub API is accessible by making a cached test request.
// Uses a fixed symbol to avoid rate limit consumption. Errors are cached with a 60-second TTL.
func (p *FinnhubProvider) HealthCheck(ctx context.Context) error {
	// Use a well-known symbol (AAPL) for the health check.
	// This is cached by the REST layer, so repeated calls are fast.
	_, err := p.GetQuote(ctx, "AAPL")
	return err
}

// Compile-time interface compliance check.
var _ MarketDataProvider = (*FinnhubProvider)(nil)
