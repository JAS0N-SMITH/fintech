# ADR 008: Snapshot-Plus-Deltas Pattern for Real-Time Market Data

## Status

Accepted

## Context

The dashboard displays live market prices for all holdings and watchlist symbols. The Go backend maintains a WebSocket connection to Finnhub's streaming API and needs to push price updates to Angular clients. There are two broad patterns:

**Option A: Full state on every update** — each WebSocket message to the client contains the full current state of all ticker prices. Simple to implement on the client; expensive on bandwidth and serialization for large portfolios.

**Option B: Snapshot + deltas** — on connect the client receives a full snapshot of all current prices. After that, only changed values (price ticks) are pushed. The client merges ticks into local state.

A third concern is reconnection: if the WebSocket drops and reconnects, the client may have missed ticks. Resuming the delta stream from a stale snapshot would show incorrect prices until the next full tick.

## Decision

Use the **snapshot-plus-deltas pattern**:

1. **On dashboard load:** the Go API makes REST calls to Finnhub's quote endpoint for each active ticker and returns a full `Quote` snapshot (price, day high, day low, open, previous close, volume, timestamp) to Angular.
2. **WebSocket stream:** after the snapshot, the Go backend pushes lightweight `PriceTick` messages (symbol, price, volume, timestamp) as Finnhub delivers them.
3. **Angular `TickerStateService`** merges incoming ticks into existing signal state: updates the current price, tracks new day high/low if the tick exceeds the snapshot values.
4. **On reconnection:** Angular re-fetches a fresh `Quote` snapshot before resuming the tick stream. This prevents a stale snapshot from accumulating incorrect ticks.

### Domain types

- **`Quote`** — full data for snapshots: symbol, price, day_high, day_low, open, previous_close, volume, timestamp
- **`PriceTick`** — lightweight streaming update: symbol, price, volume, timestamp
- **`Bar`** — historical OHLCV: symbol, open, high, low, close, volume, timestamp (used by chart views)

### Caching

Market data from Finnhub is cached in the Go service layer (in-memory with TTL):
- Real-time quotes: short TTL (5–10 seconds) to reduce duplicate API calls for the same ticker
- Historical bar data: aggressive TTL (full trading day once market closes — data is immutable)
- Never cached in Postgres — Finnhub data is ephemeral, not a source of truth

## Consequences

**Positive:**
- Bandwidth-efficient after initial load — only changed prices are pushed
- Angular state is always consistent with the snapshot baseline; no partial-state display on first render
- Reconnection re-sync prevents stale data from corrupting the accumulated delta state
- `TickerStateService` holds all state as signals, enabling fine-grained reactivity with zero extra subscriptions

**Negative:**
- Two code paths to maintain: snapshot fetch (REST) and tick merge (WebSocket)
- `TickerStateService` merge logic must be tested thoroughly — initial snapshot, tick updates, high/low tracking, reconnection resync (called out explicitly in `testing.md`)
- Go backend must buffer or drop ticks received during the snapshot fetch window to avoid a race condition
