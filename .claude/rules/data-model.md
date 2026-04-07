# Data Model Rules

## Core Principle

Transactions are the single source of truth. Current holdings, portfolio value, gain/loss, and allocation percentages are always derived — never stored. This eliminates sync bugs between stored state and transaction history.

## Entities

### Users (profiles)
Extends Supabase auth.users. Stores app-specific fields: display_name, role (user/admin), preferences (JSON), created_at, updated_at. Auth fields (email, password hash) managed by Supabase Auth.

### Portfolios
Named grouping representing a brokerage account. Fields: id, user_id, name, description, created_at, updated_at. Purely organizational — no stored financial values. A user can have multiple portfolios.

### Transactions
Every financial event. Fields: id, portfolio_id, transaction_type (buy/sell/dividend/reinvested_dividend), symbol, transaction_date, quantity (nullable — not for pure dividends), price_per_share (nullable — not for dividends), dividend_per_share (nullable — only for dividends), total_amount, notes, created_at, updated_at. Future: related_transaction_id for linking reinvested dividends to their buy.

### Watchlists
Header table. Fields: id, user_id, name, created_at, updated_at. Separate concept from portfolios — things you're interested in, not things you own.

### Watchlist Items
Individual tickers on a watchlist. Fields: id, watchlist_id, symbol, target_price (optional), notes, created_at, updated_at. Current price comes from live market data, not stored.

### Audit Log
Append-only. Fields: id, user_id, action, target_entity, target_id, before_value (JSON), after_value (JSON), ip_address, user_agent, timestamp. Never update or delete rows in this table.

## Derived Values (never stored)

- **Current quantity** = SUM(buy quantities) - SUM(sell quantities) for a symbol in a portfolio
- **Cost basis** = weighted average of buy prices (or FIFO/LIFO for tax lot tracking post-MVP)
- **Current value** = current quantity × live market price (from Finnhub)
- **Unrealized gain/loss** = current value - (current quantity × cost basis)
- **Portfolio total** = SUM of all position current values
- **Allocation %** = position current value / portfolio total × 100
- **Day high/low** = tracked in Angular TickerStateService via streaming ticks
- **Holding period** = today - transaction_date (for future tax lot feature)

## Market Data Architecture

### Provider Abstraction
Go interface defined at the service layer:
- `GetQuote(symbol) → Quote` — full quote for snapshot
- `GetHistoricalBars(symbol, timeframe, start, end) → []Bar` — historical OHLCV data
- `StreamPrices(symbols, handler func(PriceTick))` — real-time price stream

### Domain Types
- **Quote** — full data: symbol, price, day_high, day_low, open, previous_close, volume, timestamp
- **PriceTick** — lightweight streaming: symbol, price, volume, timestamp
- **Bar** — historical OHLCV: symbol, open, high, low, close, volume, timestamp

### Snapshot-Plus-Deltas Pattern
1. On dashboard load: REST call fetches full Quote for each active ticker
2. WebSocket stream pushes PriceTick updates continuously
3. Angular TickerStateService merges ticks into existing state (update price, track new high/low)
4. On WebSocket reconnection: re-fetch full Quote snapshots, then resume streaming

## Caching Strategy

- Market data from Finnhub cached in Go service layer (in-memory with TTL or Redis)
- Historical bar data: cache aggressively (immutable once market closes for that period)
- Real-time quotes: short TTL (5-10 seconds) to reduce API calls for same ticker
- Never cache in Postgres — Finnhub data is ephemeral, not our source of truth

## Future-Proofing (designed for but not built in MVP)

- Tax lot tracking: transaction date + price per share already captured
- Dividend reinvestment linking: related_transaction_id can be added to transactions
- Locale support: user preferences JSON can store locale/currency settings
- Options/crypto: transaction_type enum is extensible, symbol format can accommodate
- Materialized views: can be added for holdings aggregation if performance requires it
