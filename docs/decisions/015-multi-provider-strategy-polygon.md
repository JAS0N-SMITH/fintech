# ADR 015: Multi-Provider Strategy with Polygon.io Fallback for Historical Data

## Status

Accepted

## Context

The application needs market data from multiple sources to support different use cases:

1. **Real-time quotes** — used by portfolio dashboard and ticker detail views
2. **Historical OHLCV bars** — used by charting components and technical analysis
3. **Symbol search** — ticker discovery for watchlist and portfolio management
4. **Streaming price ticks** — real-time WebSocket updates for positions

Initially, Finnhub was the primary provider (ADR 004). However, limitations emerged:

- Finnhub free tier restricts historical bar data availability (limited date ranges, lower resolution)
- Different providers excel at different data types (Finnhub: quotes + streaming, Polygon: historical depth)
- Provider rate limits and availability vary — fallback strategy reduces user-facing failures
- Future expansion may require crypto, options, or alternative exchanges

The provider abstraction (ADR 004) was designed for extensibility but wasn't fully utilized until now.

## Decision

Implement a **multi-provider strategy** with the following architecture:

### Provider Stack

1. **Primary Provider: Finnhub**
   - GetQuote() — real-time full quotes with day_high, day_low, volume
   - StreamPrices() — WebSocket real-time price ticks
   - GetSymbols() — US stock ticker list for symbol search
   - Cached aggressively (5-10s TTL) to stay within rate limits

2. **Fallback Provider: Polygon.io**
   - GetHistoricalBars() — deep historical OHLCV data (years of history, 1-min resolution)
   - Used only when Finnhub historical request fails (not implemented, rate limited, or invalid symbol)
   - Free tier: 5 API calls/min, sufficient for on-demand chart loads
   - Does not implement GetQuote, StreamPrices, GetSymbols (placeholder stubs return errors)

### FallbackProvider Wrapper

Introduced `FallbackProvider` struct that wraps primary + optional fallback:

```go
type FallbackProvider struct {
    primary  MarketDataProvider
    fallback MarketDataProvider // optional; may be nil
}
```

Behavior:
- **GetHistoricalBars**: Tries primary first. On error (`ErrInvalidSymbol`, `ErrProviderUnavailable`), tries fallback.
  - Does NOT retry on `ErrRateLimited` — rate limit is a circuit breaker signal, not a transient error.
  - Returns fallback error if both fail.
- **All other methods** (GetQuote, StreamPrices, GetSymbols, HealthCheck): Delegate to primary only.

### SearchSymbols Endpoint

Added `SearchSymbols(ctx, query, limit)` to the `MarketDataProvider` interface:

- Returns list of supported symbols matching the query
- Filters by case-insensitive prefix match on symbol and substring match on description
- Results capped at limit (1-50; defaults to 20)
- Service layer caches entire symbol list with long TTL (1 hour) — list is relatively static
- Endpoint exposed at `GET /api/v1/market/search/symbols?query=...&limit=...`

## Consequences

### Positive

- **Resilience**: Historical chart loads succeed even if Finnhub is unavailable
- **Flexibility**: Each provider is specialized for its strength (Finnhub: real-time, Polygon: historical depth)
- **Cost-effective**: Polygon's free tier handles historical requests; Finnhub stays within free tier for real-time
- **Future-proof**: Adding providers (e.g., Alpha Vantage, IEX) is straightforward — just implement the interface
- **User experience**: Fallback mechanism means chart pages load successfully more often, even under provider degradation
- **Testability**: Provider abstraction remains; swap implementations easily for testing

### Negative

- **Increased complexity**: Two providers mean tracking health, rate limits, and error codes for each
- **Data consistency**: Finnhub and Polygon may have slight differences in historical data (timestamps, OHLC values)
  - Mitigated by: Cache historical bars aggressively; users see consistent data within cache window
- **Debugging**: Fallback silently retries — logs must clearly indicate which provider succeeded
- **API key management**: Now requires both Finnhub and Polygon API keys in environment
  - Mitigated by: .env.example template and clear config docs

### Risks

- **Rate limit cascade**: If Finnhub hits rate limit on quotes, users can't load dashboard (fallback not attempted)
  - Mitigation: Monitor Finnhub rate limit headers; add alert if approaching limit
  - Future: Consider separate rate limit budgets for quotes vs. historical
- **Silent data degradation**: If Polygon symbol doesn't exist but Finnhub does (or vice versa), user sees "not found" silently
  - Mitigated by: Clear error messages returned to client; admin can see provider health via HealthCheck endpoint
- **Polygon API contract drift**: Polygon API updates may break our struct unmarshalling
  - Mitigated by: Version tests with fixed Polygon sample responses; add integration tests against Polygon sandbox

## Implementation Notes

### Configuration

```go
// In config/config.go:
FinnhubAPIKey string // env: FINNHUB_API_KEY (required)
PolygonAPIKey  string // env: POLYGON_API_KEY (optional; falls back to Finnhub only if empty)
```

### Initialization

```go
// In cmd/api/main.go:
finnhub := provider.NewFinnhubProvider(cfg.FinnhubAPIKey)
var fallback provider.MarketDataProvider
if cfg.PolygonAPIKey != "" {
    fallback = provider.NewPolygonProvider(cfg.PolygonAPIKey)
}
wrapped := provider.NewFallbackProvider(finnhub, fallback)
svc := service.NewMarketDataService(wrapped)
```

### Error Mapping

Handler maps provider errors to HTTP responses (see ADR 011):
- `ErrInvalidSymbol` → 422 Unprocessable Entity
- `ErrRateLimited` → 429 Too Many Requests
- `ErrProviderUnavailable` → 502 Bad Gateway

## Alternatives Considered

### A. Single provider switch (Finnhub → Polygon)
- Rejected: Polygon lacks real-time streaming; Finnhub lacks historical depth. Either alone is incomplete.

### B. Cache all historical bars in Postgres at startup
- Rejected: S&P 500 stocks × 5+ years × minute resolution = millions of rows. Impractical for MVP.

### C. Try all providers in parallel
- Rejected: Unnecessary latency; primary provider is usually available. Sequential fallback is sufficient.

### D. Client-side provider selection
- Rejected: Exposes provider logic to frontend; harder to change provider strategy later. Backend abstraction is cleaner.

## Related ADRs

- **ADR 004**: Provider abstraction design rationale
- **ADR 011**: Error handling strategy (RFC 7807 Problem Details)
- **ADR 008**: Snapshot-plus-deltas pattern (uses GetQuote + StreamPrices)
