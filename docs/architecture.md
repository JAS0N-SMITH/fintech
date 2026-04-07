# Fintech Portfolio Dashboard — System Architecture

## Overview

Fintech is a web-based stock portfolio dashboard that lets users track holdings across multiple brokerage accounts, view real-time market data, and manage watchlists. The system follows a client-server architecture with a clear separation between the Angular single-page application, the Go API service, and a managed Postgres database.

This is a learning project designed to teach modern Angular (v21, signals, zoneless), Go API development with Gin, and fintech patterns including real-time data streaming, financial calculations, and security-first design.

---

## System Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        Browser                               │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              Angular v21 (Zoneless)                     │  │
│  │                                                         │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │  AuthService │  │ TickerState  │  │  Portfolio    │  │  │
│  │  │  (Signals)   │  │ Service      │  │  Service      │  │  │
│  │  │             │  │ (Signals +   │  │  (Signals)    │  │  │
│  │  │             │  │  WebSocket)  │  │              │  │  │
│  │  └──────┬──────┘  └──────┬───────┘  └──────┬───────┘  │  │
│  │         │                │                  │          │  │
│  │         │         ┌──────┴───────┐          │          │  │
│  │         │         │  RxJS        │          │          │  │
│  │         │         │  WebSocket   │          │          │  │
│  │         │         │  Connection  │          │          │  │
│  │         │         └──────┬───────┘          │          │  │
│  └─────────┼────────────────┼──────────────────┼──────────┘  │
│            │                │                  │              │
└────────────┼────────────────┼──────────────────┼──────────────┘
             │ HTTPS          │ WSS              │ HTTPS
             │ (JWT)          │ (JWT)            │ (JWT)
             ▼                ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│                     Go / Gin API                             │
│                                                              │
│  Middleware Chain:                                            │
│  RequestID → Logger → Recovery → CORS → RateLimit            │
│                         │                                    │
│              ┌──────────┴──────────┐                         │
│              │                     │                         │
│         Public Routes        Protected Routes                │
│         /auth/*              Auth Middleware                  │
│                              │           │                   │
│                         User Routes   Admin Routes           │
│                         /portfolios   RequireRole("admin")   │
│                         /transactions AuditLog Middleware     │
│                         /watchlists   /admin/*               │
│                              │                               │
│  ┌───────────────────────────┼─────────────────────────┐     │
│  │                    Service Layer                     │     │
│  │  Defines interfaces it depends on:                   │     │
│  │  - PortfolioRepository                               │     │
│  │  - TransactionRepository                             │     │
│  │  - MarketDataProvider                                │     │
│  └──────────┬────────────────────────────┬─────────────┘     │
│             │                            │                   │
│  ┌──────────▼──────────┐    ┌────────────▼──────────────┐    │
│  │  Repository Layer   │    │   Provider Layer           │    │
│  │  (pgx → Postgres)   │    │   (Finnhub REST + WS)     │    │
│  │                     │    │                            │    │
│  │  - PortfolioRepo    │    │   ┌────────────────────┐   │    │
│  │  - TransactionRepo  │    │   │ MarketDataProvider │   │    │
│  │  - WatchlistRepo    │    │   │ interface           │   │    │
│  │  - AuditLogRepo     │    │   ├────────────────────┤   │    │
│  │                     │    │   │ FinnhubProvider     │   │    │
│  │                     │    │   │ (current)           │   │    │
│  │                     │    │   ├────────────────────┤   │    │
│  │                     │    │   │ AlpacaProvider     │   │    │
│  │                     │    │   │ (future)           │   │    │
│  │                     │    │   ├────────────────────┤   │    │
│  │                     │    │   │ MockProvider       │   │    │
│  │                     │    │   │ (testing)          │   │    │
│  └──────────┬──────────┘    │   └────────────────────┘   │    │
│             │               └────────────┬───────────────┘    │
└─────────────┼────────────────────────────┼────────────────────┘
              │                            │
              ▼                            ▼
┌──────────────────────┐     ┌──────────────────────────┐
│   Supabase Postgres  │     │       Finnhub API        │
│                      │     │                          │
│  - profiles          │     │  REST: /quote, /candle   │
│  - portfolios        │     │  WebSocket: streaming    │
│  - transactions      │     │  ticks                   │
│  - watchlists        │     │                          │
│  - watchlist_items   │     │  Free tier:              │
│  - audit_log         │     │  60 req/min              │
│                      │     │  30 WS connections       │
│  Auth: Supabase Auth │     │                          │
│  RLS: defense layer  │     │  Upgrade path:           │
│                      │     │  Alpaca, Polygon         │
└──────────────────────┘     └──────────────────────────┘
```

---

## Data Flow Patterns

### Authentication Flow

1. User submits credentials in Angular login form
2. Angular AuthService calls Supabase Auth directly (not through Go API)
3. Supabase returns access token (JWT, 15-min TTL) and refresh token (7-day TTL)
4. Access token stored in AuthService signal (memory only)
5. Refresh token stored in HTTP-only, Secure, SameSite=Strict cookie
6. Angular HTTP interceptor attaches access token to all Go API requests
7. Go auth middleware validates JWT signature, expiration, and extracts user ID and role
8. On token expiry, Angular silently requests new access token using refresh token
9. On refresh token expiry, user is redirected to login

### Real-Time Market Data (Snapshot-Plus-Deltas)

1. User opens dashboard — Angular TickerStateService identifies active tickers from portfolio holdings
2. TickerStateService calls Go API REST endpoint for full Quote snapshot per ticker
3. Go API checks in-memory cache (TTL-based) — cache hit returns cached data, cache miss fetches from Finnhub REST API, caches result, returns to Angular
4. TickerStateService stores snapshot in a signal per ticker (full state: price, dayHigh, dayLow, open, previousClose, volume)
5. TickerStateService opens WebSocket connection to Go API, subscribes to active symbols
6. Go WebSocket relay maintains a persistent connection to Finnhub WebSocket, fans out PriceTick messages to connected Angular clients
7. On each PriceTick arrival, TickerStateService merges into existing signal state:
   - Price always updates
   - dayHigh updates only if new price > current dayHigh
   - dayLow updates only if new price < current dayLow
8. Angular components read ticker signals — UI updates automatically via zoneless change detection
9. On WebSocket disconnect: state transitions to "reconnecting", exponential backoff retry begins
10. On reconnect: full Quote snapshots re-fetched for all active tickers, then streaming resumes

### Portfolio Value Calculation

All financial values are derived, never stored:

1. **Current holdings** = for each symbol in a portfolio, SUM(buy quantities) - SUM(sell quantities) from transactions table
2. **Cost basis per share** = weighted average of buy transaction prices
3. **Current value per position** = current quantity × live price (from TickerStateService signal)
4. **Unrealized gain/loss per position** = current value - (current quantity × cost basis)
5. **Portfolio total value** = SUM of all position current values
6. **Portfolio day change** = SUM of (current quantity × (current price - previous close)) per position
7. **Allocation percentage** = position current value / portfolio total × 100

These calculations live in Angular as computed signals. They recalculate automatically when either transaction data changes (user adds a trade) or live price ticks arrive (TickerStateService updates).

---

## Security Architecture

### Defense-in-Depth (Three Authorization Layers)

```
Request → Angular Guard → Go Middleware → Supabase RLS
            (UX)           (Enforcement)    (Data layer)
```

1. **Angular guards** prevent navigation and code download (canMatch for admin). UX convenience only — not a security boundary.
2. **Go middleware** enforces authorization on every API request. Validates JWT, checks role, returns 403 for unauthorized access. This is the primary security boundary.
3. **Supabase Row Level Security** enforces data isolation at the database level. Users can only query their own data even if the API layer has a bug. Defense-in-depth backstop.

### RBAC Model

Two roles for MVP:
- **user** — full access to own portfolios, transactions, watchlists. No access to admin routes.
- **admin** — everything a user can do plus: manage other users, view audit logs, monitor system health, manage feature flags.

Roles stored in profiles table. JWT claims include role. Go middleware checks role on protected route groups. Angular guards use role from AuthService signal to control navigation and UI element visibility.

### Sensitive Data Handling

- Passwords: managed entirely by Supabase Auth (bcrypt hashing, never touches Go API)
- JWTs: access token in memory only, refresh token in HTTP-only cookie
- API keys (Finnhub, Supabase): environment variables, never in code or config files
- Financial data: not classified as PII for this app (no real money, no account numbers), but still not logged in plaintext
- Audit log: masks PII fields before storage, append-only table

---

## Technology Decisions Summary

| Decision | Choice | Rationale |
|---|---|---|
| Frontend framework | Angular v21 | Signals-first, zoneless, strong opinions for large apps, learning goal |
| Backend framework | Go / Gin | Concurrent request handling, compiled binary, learning goal |
| Database | Supabase (Postgres) | Managed hosting, built-in auth, familiar from Grooping project |
| Market data | Finnhub | Best free tier for real-time data, WebSocket support, provider-abstracted |
| Financial charts | TradingView Lightweight Charts | Purpose-built for financial data, 40KB, 50K+ candles performant |
| General charts | Chart.js (ng2-charts) | Pie/doughnut/scatter for allocation views, ~67KB, well-supported |
| Component library | PrimeNG | Extensive component set, strong data tables, built-in ARIA, prior experience |
| CSS framework | Tailwind CSS | Utility-first for layout, pairs well with PrimeNG, minimal setup in Angular |
| Unit testing (Angular) | Vitest | Angular 21 default, stable, fast |
| E2E testing | Playwright | Cross-browser, lower memory than Cypress, visual regression built-in |
| Unit testing (Go) | stdlib testing | Table-driven tests, httptest, no external framework needed |
| Integration testing (Go) | testcontainers-go | Real Postgres in tests, reliable, clean teardown |
| DB driver | pgx | Direct Postgres access, 30-50% faster than GORM, no ORM abstraction |
| Migrations | Goose | Go embedding support, bidirectional SQL migrations |
| Auth | Supabase Auth + JWT | Managed auth on frontend, stateless validation on backend |
| Logging | slog (Go stdlib) | Structured JSON, zero dependencies, correlation IDs via context |
| API docs | swaggo/swag | Generates Swagger from handler annotations, serves Swagger UI |

---

## Deployment Architecture (Future)

Not in MVP scope, but the system is designed to support:

```
                    ┌─────────────┐
                    │  Cloudflare  │
                    │  (CDN + WAF) │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │                         │
    ┌─────────▼─────────┐   ┌──────────▼──────────┐
    │  Angular Static    │   │   Go API Container   │
    │  (S3 / Vercel /   │   │   (Fly.io / Railway   │
    │   Cloudflare      │   │    / Cloud Run)       │
    │   Pages)          │   │                       │
    └───────────────────┘   └──────────┬────────────┘
                                       │
                            ┌──────────▼──────────┐
                            │  Supabase Postgres   │
                            │  (managed)           │
                            └─────────────────────┘
```

Angular builds to static files — deployable to any CDN or static host. Go compiles to a single binary — deployable to any container platform. Supabase is already managed. No infrastructure to maintain beyond the Go service.

---

## Key Architectural Patterns

**Transactions as source of truth** — Holdings, portfolio values, and allocation percentages are always derived from transaction records. No denormalized holdings table. This eliminates sync bugs at the cost of computation, which is negligible at expected scale. Materialized views can be added later if needed.

**Provider abstraction** — Market data access goes through a Go interface defined at the service layer. FinnhubProvider is the current implementation. Swapping to Alpaca or Polygon means writing a new struct with the same methods. No changes to services, handlers, or Angular code.

**Snapshot-plus-deltas** — Dashboard loads full quote data via REST, then WebSocket ticks update only what changes. Reconnection re-fetches snapshots before resuming stream. This is the standard pattern used by professional trading platforms.

**Signals for state, RxJS for streams** — Angular signals manage all synchronous UI state. RxJS manages the WebSocket connection lifecycle and stream transformation. Signals and RxJS meet at the service boundary via toSignal() or manual .set() calls in subscriptions.

**Defense-in-depth** — Authorization enforced at three layers (Angular guards, Go middleware, Supabase RLS). Any single layer failing still leaves two layers protecting data. Security headers, rate limiting, input validation, and audit logging reinforce the boundaries.
