# Phase 8: Watchlists — Build Summary

**Date:** 2026-04-11
**Duration:** Phase 8 of 12
**Branch:** main

---

## What Was Built

Phase 8 added the ability to create personal watchlists for research and tracking. Unlike portfolios (things you own), watchlists are curated collections of tickers you're interested in, with optional target price alerts.

### Frontend (Angular)

| Component | Description |
|-----------|-------------|
| `WatchlistListComponent` (`features/watchlist/pages/watchlist-list.component.ts`) | Lists all user watchlists in a card grid; allows create, edit, delete |
| `WatchlistDetailComponent` (`features/watchlist/pages/watchlist-detail.component.ts`) | Displays watchlist items (symbols) with live prices, target price, and notes |
| `WatchlistService` (`features/watchlist/services/watchlist.service.ts`) | CRUD service for watchlists and items; manages state as signals |
| `WatchlistItemsTableComponent` | Table showing: Symbol, Current Price (live), Target Price (user-set), Change %, Notes; add/remove item buttons |
| `CreateWatchlistDialogComponent` | Form dialog for creating a new watchlist (name, description) |
| `EditWatchlistDialogComponent` | Form dialog for editing watchlist metadata |
| Routes | `/watchlist`, `/watchlist/:id` — lazy-loaded feature routes |

**State Management:**
- `watchlists: Signal<Watchlist[]>` — reactive list of all watchlists
- Per-watchlist live prices fetched via `TickerStateService.subscribe()` for each symbol in items
- Target price stored in database; shown alongside live price for comparison

### Backend (Go)

| Component | Description |
|-----------|-------------|
| `model/Watchlist`, `WatchlistItem` | Domain types: watchlist (id, user_id, name, description); item (id, watchlist_id, symbol, target_price, notes) |
| `handler/watchlist.go` | HTTP handlers for watchlist CRUD; item add/remove/update |
| `service/watchlist.go` | Business logic; user isolation; validates symbol format |
| `repository/watchlist.go` | pgx queries; atomic operations for adding/removing items |
| `middleware/watchlist.go` | User isolation — only users can access their own watchlists |
| Routes | `GET/POST /api/v1/watchlists`, `GET/PUT/DELETE /api/v1/watchlists/:id`, CRUD on items |

**Key Decisions:**
- Watchlist items are **not** transactions — just metadata pointers to symbols
- Target price is nullable (optional alert threshold)
- Current price always fetched from live market data, never stored
- User isolation enforced at query level (WHERE user_id = $1)

### Database

**New Migration: `00005_create_watchlists_and_audit_log.sql` (partial)**
```sql
CREATE TABLE watchlists (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES auth.users(id),
  name TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

CREATE TABLE watchlist_items (
  id UUID PRIMARY KEY,
  watchlist_id UUID NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
  symbol TEXT NOT NULL,
  target_price NUMERIC(10,2),
  notes TEXT,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- RLS policies ensure users can only access their own watchlists
```

---

## Architecture Decisions Applied

- **ADR 004** — Live prices for watchlist items sourced from `MarketDataProvider` (Finnhub)
- **ADR 007** — Current price for watchlist items is **never stored** — always derived from live market data
- **ADR 008** — Snapshot-plus-deltas used when rendering item list: fetch Quote snapshots on page load, merge WebSocket ticks as they arrive
- **security.md** — User isolation at database level via RLS; symbol validation (regex); audit logging for watchlist create/delete

---

## Test Coverage Added

| Test file | Tests | Notes |
|-----------|-------|-------|
| `watchlist.service.spec.ts` | 9 | TDD; create, read, delete; user isolation |
| `watchlist-detail.component.spec.ts` | 7 | Item display, live price updates, target price |
| `watchlist-list.component.spec.ts` | 5 | List rendering, create/delete dialogs |
| `handler/watchlist_test.go` | 11 | CRUD endpoints, user isolation, symbol validation |
| `repository/watchlist_test.go` | 8 | Database queries, cascading deletes |
| `service/watchlist_test.go` | 6 | Business logic, error handling |

**Total new tests: 46** across frontend and backend.

---

## Security Enhancements

**Audit Logging:**
- Watchlist create/delete logged to `audit_log` table with action and timestamps
- Symbol validation: alphanumeric, dots, hyphens only (regex `^[A-Z0-9.-]+$`)
- No PII in watchlist names or notes — just free-form user text

---

## UI/UX Features

**Responsive Design:**
- Card grid for watchlist list adapts to screen size (1 column mobile, 3+ desktop)
- Item table scrollable on narrow screens

**Accessibility (WCAG 2.1 AA):**
- Add/Remove item buttons keyboard navigable
- Target price input has aria-label
- Price color-coded (green gain, red loss) with text fallback ("↑", "↓")

---

## Known Limitations

- No bulk operations (add/remove multiple items at once)
- No watchlist sharing between users
- No watchlist sorting/filtering (planned for later)
- No price alerts/notifications (deferred to notifications phase)
- No export to CSV (deferred)

---

## How to Verify

### Frontend

```bash
cd frontend && ng serve
# Navigate to Watchlists
# Create a watchlist ("Tech Stocks")
# Add items (AAPL, MSFT, TSLA)
# Observe live prices update as ticks arrive
# Set target prices and notes
# Delete watchlist — confirms delete action
```

### Backend

```bash
cd backend && make dev

# Create watchlist
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tech","description":""}' \
  http://localhost:8080/api/v1/watchlists

# Add item
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"symbol":"AAPL","target_price":150}' \
  http://localhost:8080/api/v1/watchlists/{id}/items

# List items with live prices
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/watchlists/{id}
```

### Tests

```bash
cd frontend && ng test --watch=false   # watchlist tests pass ✅
cd backend && make test         # watchlist tests pass ✅
```

---

## Next Phase

Phase 9: Connection State & Error Handling — Resilient WebSocket reconnection, user-friendly error messages.
