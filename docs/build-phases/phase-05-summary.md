# Phase 5: Market Data Integration — Build Summary

**Dates:** 2026-04-11
**Branch:** main
**Commits:** ea54802 → 348a6d7

---

## What Was Built

Phase 5 wired live Finnhub market data into the portfolio dashboard. Holdings now display real-time prices, current values, and unrealised gain/loss that update automatically as WebSocket ticks arrive.

### Backend (Go)

| Component | Description |
|-----------|-------------|
| `model.Quote`, `PriceTick`, `Bar`, `Timeframe` | Domain types for market data (no vendor coupling) |
| `provider.MarketDataProvider` | Interface decoupling services from any specific data source |
| `provider.FinnhubProvider` | REST (quotes + bars) + WebSocket (price stream) via gorilla/websocket |
| `provider.MockProvider` | Configurable test double for unit tests |
| `service.MarketDataService` | Thread-safe in-memory cache: 10s TTL for quotes, 24h for historical bars |
| `handler.MarketDataHandler` | `GET /api/v1/quotes/:symbol`, `GET /api/v1/quotes` (batch), `GET /api/v1/bars/:symbol` |
| `handler.WebSocketHandler` | `GET /api/v1/ws/prices` — authenticated relay with per-client subscribe/unsubscribe |
| `config.Config` | `FinnhubAPIKey`, `FinnhubBaseURL`, `FinnhubWSURL` with startup validation |

### Frontend (Angular)

| Component | Description |
|-----------|-------------|
| `market-data.model.ts` | `Quote`, `PriceTick`, `Bar`, `Timeframe`, `ConnectionState`, `TickerState` interfaces |
| `Holding` (extended) | Added `currentPrice`, `currentValue`, `gainLoss`, `gainLossPercent` (null until prices arrive) |
| `MarketDataService` | REST HTTP service: `getQuote`, `getQuotesBatch`, `getHistoricalBars` |
| `TickerStateService` | Snapshot-plus-deltas (ADR 008): REST fetch → WebSocket stream, signal state, reconnection with exponential backoff |
| `enrichHoldingsWithPrices()` | Pure function: computes financial fields from quantity × price, exported for TDD |
| `TransactionService.holdings` | Computed signal now merges `TickerStateService.tickers()` — auto-updates on price or transaction change |
| `HoldingsTableComponent` | New columns: Price, Value, Gain/Loss with loading skeletons; connection state badge (WCAG 2.1 AA) |
| `PortfolioDetailComponent` | Portfolio total and total gain/loss summary bar; passes `connectionState` to table |

---

## Architecture Decisions Applied

- **ADR 004** — FinnhubProvider as concrete implementation behind `MarketDataProvider` interface; swappable without service changes
- **ADR 008** — Snapshot-plus-deltas: REST Quote snapshot on subscribe, WebSocket PriceTick stream for updates; re-fetch snapshots on reconnect before resuming ticks
- **ADR 007** — Transactions as source of truth: holdings still derived, now enriched with live prices in the computed signal
- **security.md** — JWT validation on WebSocket upgrade; symbol input validation (regex); never log financial data

---

## Test Coverage Added

| Test file | Tests | Notes |
|-----------|-------|-------|
| `provider/finnhub_test.go` | 12 | httptest.NewServer mocks; error path coverage |
| `provider/mock_test.go` | 3 | MockProvider correctness |
| `service/market_data_test.go` | 11 | TDD; cache hit/miss/expiry, error mapping, concurrency |
| `handler/market_data_test.go` | 11 | Symbol validation, timeframe validation, date parsing |
| `market-data.service.spec.ts` | 5 | HttpTestingController; all endpoints |
| `ticker-state.service.spec.ts` | 11 | TDD; snapshot init, tick merge, high/low tracking, reconnect resync |
| `transaction.service.spec.ts` | +8 | TDD for `enrichHoldingsWithPrices()`: gain, loss, zero cost, multi-symbol |

**Total new tests: 61** across backend and frontend.

---

## Pre-existing Issues (not introduced in Phase 5)

The backend `internal/handler` transaction handler tests (14 cases) were already failing before Phase 5 work began — a routing mismatch from the route refactor in commit `092a1a3`. These are unchanged by Phase 5.

---

## How to Verify

### Backend smoke test (requires `FINNHUB_API_KEY` in `.env`)

```bash
cd backend && make dev   # start server

# REST quote
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/quotes/AAPL

# Historical bars
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/bars/AAPL?timeframe=1M"

# WebSocket (requires wscat: npm i -g wscat)
wscat -c "ws://localhost:8080/api/v1/ws/prices" \
  -H "Authorization: Bearer $TOKEN"
# send: {"action":"subscribe","symbols":["AAPL"]}
# expect: stream of {"type":"tick","data":{...}}
```

### Frontend

```bash
cd frontend && ng serve   # open http://localhost:4200
# Navigate to a portfolio with holdings
# Holdings table shows — skeletons initially
# After WebSocket connects → prices populate, gain/loss shows with colour coding
# Disconnect network → badge shows "Reconnecting…"
# Restore → badge shows "Live", prices resync
```

### Tests

```bash
cd backend  && make test        # new packages: provider ✅ service ✅
cd frontend && ng test --watch=false   # 76/76 ✅
```
