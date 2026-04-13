# Phase 7: Ticker Detail View — Build Summary

**Date:** 2026-04-11
**Duration:** Phase 7 of 12
**Branch:** main

---

## What Was Built

Phase 7 added an individual stock detail page showing real-time price, interactive candlestick charts with technical analysis tools, and historical performance metrics. This feature enables deep dives into individual holdings and research.

### Frontend (Angular)

| Component | Description |
|-----------|-------------|
| `TickerDetailComponent` (`features/tickers/pages/ticker-detail.component.ts`) | Main page; displays quote header, price/change, interactive TradingView chart, historical metrics |
| `TickerHeaderComponent` | Header card showing current price, 24h change %, day high/low, volume, market cap |
| `TickerChartComponent` | TradingView Lightweight Charts integration; candlestick view with toolbar (timeframes: 1D/1W/1M/3M/1Y) |
| `DayChangeComponent` | Displays intraday price change with color coding (green/red); updated via WebSocket ticks |
| `TickerStateService` | Manages real-time ticker state: snapshot fetch + WebSocket delta updates; tracks day high/low from ticks |
| Routes | `/tickers/:symbol` — lazy-loaded feature route with symbol routing param |

**Architecture Decisions:**
- TradingView Lightweight Charts for candlestick rendering (lightweight, no licensing cost for educational use)
- Real-time day high/low tracked in `TickerStateService.tickers()` signal (computed from tick stream)
- Timeframe switching triggers REST fetch of bars from backend
- On chart mount: fetch full 1Y of historical data, then stream current day's ticks

### Backend (Go)

**Enhanced from Phase 5:**
- `GetHistoricalBars(symbol, timeframe, start, end)` provider method
- Finnhub REST endpoint `/quote` and `/candle` cached in service (24h TTL for historical, 10s for intraday)
- `MarketDataHandler` extended with `/api/v1/bars/:symbol?timeframe=1M` endpoint
- Bar domain type: OHLCV + timestamp, normalized from Finnhub format

### Integration

**Data Flow:**
1. User navigates to `/tickers/AAPL`
2. `TickerDetailComponent` calls `TickerStateService.subscribe('AAPL')`
3. Service fetches Quote snapshot + 1Y of historical bars via REST
4. Component renders header with snapshot data, chart with bars
5. WebSocket ticks for AAPL arrive continuously
6. Service merges ticks into `tickers()` signal; updates currentPrice, day high/low
7. Chart rerenders; header updates reactively

---

## Architecture Decisions Applied

- **ADR 004** — Finnhub provider supplies both REST bars and WebSocket ticks; abstraction maintained
- **ADR 005** — TradingView Lightweight Charts chosen for financial-grade candlestick rendering
- **ADR 008** — Snapshot-plus-deltas: fetch full 1Y bars on subscribe, then stream intraday ticks
- **ADR 007** — Day high/low computed from tick stream, not stored in database

---

## Test Coverage Added

| Test file | Tests | Notes |
|-----------|-------|-------|
| `market-data.service.spec.ts` | +5 | Bars endpoint, timeframe validation |
| `ticker-state.service.spec.ts` | +6 | Day high/low tracking from ticks, timeframe changes |
| `ticker-detail.component.spec.ts` | 8 | Route params, chart rendering, real-time updates |
| `handler/market_data_test.go` | +5 | Bars endpoint, timeframe enum validation |
| `provider/finnhub_test.go` | +4 | Historical bars fetch, date range handling |

**Total new tests: 28** across frontend and backend.

---

## UI Enhancements

**Responsive Design:**
- Chart container responsive to viewport (uses TradingView library's auto-sizing)
- Header metrics stack on narrow screens (mobile deferred per ADR 010, but responsive prep done)

**Accessibility (WCAG 2.1 AA):**
- Chart has `aria-label="Candlestick chart for {symbol} price movement"`
- Timeframe buttons have keyboard nav
- High/low indicators have sufficient color contrast

---

## Known Limitations

- No technical indicators (MA, RSI, Bollinger Bands) — use TradingView library's built-in toolbar
- No intraday minute-level data — lowest granularity is 1-day bars
- No trade annotations (buy/sell markers on chart) — deferred to later phase
- Volume data from Finnhub cached but not visualized (volume bar chart deferred)

---

## How to Verify

### Frontend

```bash
cd frontend && ng serve
# Navigate to Portfolio > Holdings > click a symbol (e.g., "AAPL")
# Ticker detail page loads with chart
# Switch timeframes (1D, 1W, 1M, etc.)
# Observe price update in header as WebSocket ticks arrive
```

### Backend

```bash
cd backend && make dev

# Fetch 1Y of bars for AAPL
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/bars/AAPL?timeframe=1M"

# Fetch custom date range
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/bars/AAPL?timeframe=1D&start=2025-01-01&end=2026-04-12"
```

### Tests

```bash
cd frontend && ng test --watch=false   # ticker tests pass ✅
cd backend && make test         # bars tests pass ✅
```

---

## Next Phase

Phase 8: Watchlists — Add favorites feature for research tracking.
