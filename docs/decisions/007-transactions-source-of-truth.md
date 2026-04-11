# ADR 007: Transactions as Single Source of Truth — No Holdings Table

## Status

Accepted

## Context

The application needs to display current portfolio holdings: how many shares of each symbol the user owns, the average cost basis, unrealized gain/loss, and portfolio allocation percentages. There are two broad approaches to storing this data:

**Option A: Derived-on-read** — store only transactions; calculate holdings at query time by aggregating transaction history.

**Option B: Materialized holdings** — maintain a separate `holdings` table (or `positions` table) that is updated on every transaction write. Holdings are pre-computed and read directly.

The core trade-off is between write complexity and read complexity.

## Decision

**Store only transactions. Never store derived financial values in the database.**

Current quantity, cost basis, unrealized gain/loss, portfolio total, and allocation percentages are always computed from the transaction log:

- **Current quantity** = `SUM(buy + reinvested_dividend quantities) - SUM(sell quantities)` for a symbol
- **Cost basis** = weighted average of buy prices
- **Current value** = current quantity × live market price (from Finnhub — never stored)
- **Unrealized gain/loss** = current value − (current quantity × cost basis)
- **Portfolio total** = sum of all position current values
- **Allocation %** = position value / portfolio total × 100

The `QuantityHeld` aggregate query in `repository/transaction.go` is the canonical implementation of this derivation at the database level.

## Consequences

**Positive:**
- **No sync bugs.** A materialized holdings table must be kept consistent with transactions on every write. Any bug in that sync logic (failed transaction, concurrent writes, missed update) silently corrupts the user's displayed balance — a serious problem in a financial application.
- **Full audit trail.** Every state change is captured as an immutable transaction record. Holdings at any point in time can be reconstructed by replaying transactions up to that date.
- **Correct tax lot support.** Cost basis calculations (FIFO, LIFO, specific lot) are possible in future because individual purchase records are preserved.
- **Simpler write path.** `CREATE transaction` is a single insert with no dependent updates.
- **Natural dividend support.** Cash dividends have no quantity change; they're recorded as `dividend` type transactions with `dividend_per_share`, keeping the ledger complete.

**Negative:**
- **Read cost.** Displaying holdings requires aggregating all transactions for a symbol, not a single row lookup. For portfolios with thousands of transactions this could be slow — mitigated by indexes on `(portfolio_id, symbol)` and Postgres materialized views if needed post-MVP.
- **No single "current balance" row to display.** Every holdings view requires a query join + aggregation.

**Future escape hatch:**
If read performance becomes a problem, Postgres materialized views can be added as a caching layer without changing the application's source-of-truth model. The transactions table remains authoritative; the materialized view is just an index.
