# ADR 004: Finnhub as Primary Market Data Provider with Provider Abstraction

## Status

Accepted

## Context

The dashboard needs real-time and historical stock market data including:

- Current quotes (price, day high/low, volume, previous close)
- Historical OHLCV bars for charting (1D through ALL timeframes)
- Real-time streaming price ticks via WebSocket

Market data providers vary in pricing, rate limits, data quality, and API design. We need the ability to switch providers without rewriting business logic.

## Decision

Use Finnhub as the MVP market data provider, accessed through a Go interface:

```go
type MarketDataProvider interface {
    GetQuote(ctx context.Context, symbol string) (*Quote, error)
    GetHistoricalBars(ctx context.Context, symbol string, timeframe Timeframe, start, end time.Time) ([]Bar, error)
    StreamPrices(ctx context.Context, symbols []string, handler func(PriceTick)) error
}
```

- **FinnhubProvider** implements this interface for MVP
- Future providers (Alpaca, Polygon) implement the same interface
- **MockProvider** implements it for testing
- Go service layer caches quotes in-memory with TTL to reduce API calls

## Consequences

**Positive:**
- Finnhub's free tier (60 req/min, 30 WebSocket connections) is sufficient for development
- Provider interface makes swapping data sources a single struct replacement
- In-memory caching reduces API call volume and improves response times
- WebSocket support enables real-time price streaming without polling

**Negative:**
- Free tier rate limits require careful caching strategy
- Finnhub's data quality on the free tier may have delays or gaps
- 30 WebSocket connection limit constrains the number of simultaneous streaming symbols

**Risks:**
- If Finnhub deprecates their free tier, we can implement AlpacaProvider without changing services or handlers
