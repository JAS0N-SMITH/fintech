# Stock Portfolio Dashboard

A full-stack fintech learning project for tracking brokerage account positions,
watchlists, and real-time market data.

**Stack:** Angular 21 (zoneless, signals-first) | Go/Gin | Supabase Postgres | Finnhub

> This is a learning project. The codebase is intentionally documented in depth —
> see `docs/decisions/` for Architecture Decision Records and `docs/build-phases/`
> for phase-by-phase build summaries.

**License:** Apache 2.0 — see [LICENSE](./LICENSE)

---

## Tech Stack

| Layer           | Technology                                              |
|-----------------|---------------------------------------------------------|
| Frontend        | Angular 21.2.6, zoneless + signals                      |
| Component lib   | PrimeNG 21.1.5 (Aura theme)                             |
| Styling         | Tailwind CSS v4                                         |
| Charting        | TradingView Lightweight Charts, Chart.js via ng2-charts |
| Backend         | Go 1.26.1, Gin 1.12.0                                   |
| Database driver | pgx v5.9.1, shopspring/decimal                          |
| Database        | Supabase (managed Postgres)                             |
| Migrations      | Goose v3                                                |
| Auth            | Supabase Auth (frontend) + JWT validation (Go)          |
| Market data     | Finnhub REST + WebSocket                                |
| Frontend tests  | Vitest (unit), Playwright (E2E)                         |
| Backend tests   | Go stdlib testing + testcontainers-go                   |
| API docs        | Swagger (swaggo/swag) at `/swagger/index.html`          |

---

## Architecture Overview

```
Browser
  └─ Angular 21 (signals, zoneless)
       ├─ Supabase JS client — auth only (login, token refresh)
       └─ HTTP interceptor — attaches JWT to every API request
            │
            ▼
       Go / Gin API
            ├─ JWT middleware — validates Supabase-issued tokens
            ├─ Handler layer — parses requests, maps errors to HTTP responses
            ├─ Service layer — business logic, no DB knowledge
            ├─ Repository layer — pgx queries, parameterized only
            └─ Provider layer — Finnhub REST + WebSocket relay
                     │
                     ▼
              Supabase (Postgres)
              5 migrations — profiles, portfolios, transactions,
              watchlists, audit_log
```

### Key Architectural Decision: Transactions as Source of Truth

Holdings are **never stored**. Every holding quantity, average cost basis, and
unrealized gain/loss is computed at runtime from the raw transaction ledger via
the `deriveHoldings()` pure function in the Angular service layer and equivalent
logic in the Go service layer.

This preserves full audit history, makes corrections trivial (edit the
transaction, recalculate), and avoids synchronization bugs between stored and
derived state.

See `docs/decisions/007-transactions-source-of-truth.md` for the full rationale.

---

## Getting Started

### Prerequisites

| Tool         | Minimum version | Notes                              |
|--------------|-----------------|------------------------------------|
| Node.js      | 22.x            | For Angular CLI and npm            |
| Go           | 1.26.1          |                                    |
| Angular CLI  | 21.x            | `npm install -g @angular/cli`      |
| Supabase CLI | latest          | For local migrations (optional)    |
| Docker       | any             | Required for backend integration tests |

### Environment Setup

**Backend — `backend/.env`**

```
PORT=8080
GIN_MODE=debug
DATABASE_URL=postgresql://postgres:[password]@[host]:5432/postgres
SUPABASE_URL=https://[project-ref].supabase.co
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...
JWT_SECRET=...
FINNHUB_API_KEY=...
```

Copy `backend/.env.example` and fill in values from your Supabase project settings
and your Finnhub account at `https://finnhub.io`.

**Frontend — `frontend/src/environments/environment.development.ts`**

```typescript
export const environment = {
  production: false,
  supabaseUrl: 'https://[project-ref].supabase.co',
  supabaseAnonKey: '...',
  apiBaseUrl: 'http://localhost:8080',
};
```

The frontend does **not** use a `.env` file — Angular environments are configured
in `src/environments/`.

### Run the App

```bash
# Terminal 1 — backend
cd backend && go run cmd/api/main.go

# Terminal 2 — frontend
cd frontend && ng serve
```

- Frontend: http://localhost:4200
- Backend: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html

### Run Database Migrations

Migrations are managed with Goose and live in `backend/migrations/`. They run
automatically on startup via the Go API. To run them manually:

```bash
cd backend
goose -dir migrations postgres "$DATABASE_URL" up
```

---

## API Endpoints

Full interactive documentation is available at `/swagger/index.html` when the
backend is running.

### Health

| Method | Path      | Auth | Description       |
|--------|-----------|------|-------------------|
| GET    | /health   | No   | Service health check |

### Portfolios

| Method | Path                  | Auth | Description                     |
|--------|-----------------------|------|---------------------------------|
| GET    | /api/v1/portfolios       | JWT  | List portfolios for current user |
| POST   | /api/v1/portfolios       | JWT  | Create a portfolio              |
| GET    | /api/v1/portfolios/:id   | JWT  | Get a single portfolio          |
| PUT    | /api/v1/portfolios/:id   | JWT  | Update portfolio metadata       |
| DELETE | /api/v1/portfolios/:id   | JWT  | Delete a portfolio              |

### Transactions

| Method | Path                                           | Auth | Description                          |
|--------|------------------------------------------------|------|--------------------------------------|
| GET    | /api/v1/portfolios/:id/transactions            | JWT  | List transactions for a portfolio    |
| POST   | /api/v1/portfolios/:id/transactions            | JWT  | Record a new transaction (buy/sell/dividend) |
| PUT    | /api/v1/portfolios/:id/transactions/:txid      | JWT  | Update a transaction                 |
| DELETE | /api/v1/portfolios/:id/transactions/:txid      | JWT  | Delete a transaction                 |

All protected endpoints require a `Bearer` token in the `Authorization` header.
The token is the Supabase access token obtained at login.

---

## Project Structure

```
.
├── frontend/
│   └── src/app/
│       ├── core/
│       │   ├── auth.service.ts          # Supabase auth, token signal
│       │   ├── auth.guard.ts            # Route protection
│       │   └── auth.interceptor.ts      # JWT attachment to HTTP requests
│       ├── features/
│       │   ├── auth/                    # Login, register pages
│       │   ├── portfolio/               # Portfolio list, detail, transaction form
│       │   │   ├── pages/
│       │   │   │   ├── portfolio-list/
│       │   │   │   └── portfolio-detail/
│       │   │   └── services/
│       │   │       ├── portfolio.service.ts
│       │   │       └── transaction.service.ts  # includes deriveHoldings()
│       │   ├── dashboard/               # (Phase 6 — not yet built)
│       │   ├── watchlist/               # (Phase 8 — not yet built)
│       │   └── admin/                   # (Phase 10 — not yet built)
│       ├── app.routes.ts
│       └── app.config.ts               # Zoneless bootstrap, provideHttpClient
│
├── backend/
│   ├── cmd/api/
│   │   └── main.go                     # Entry point, server startup
│   └── internal/
│       ├── handler/                    # HTTP layer — portfolio.go, transaction.go
│       ├── service/                    # Business logic — portfolio.go, transaction.go
│       ├── repository/                 # pgx queries — portfolio.go, transaction.go
│       ├── middleware/                 # JWT auth, CORS, request ID, logging
│       ├── model/                      # Domain types
│       ├── config/                     # Viper config loading
│       └── provider/                   # (Finnhub integration — Phase 5)
│
├── backend/migrations/
│   ├── 00001_create_profiles.sql
│   ├── 00002_prevent_role_escalation.sql
│   ├── 00003_create_portfolios.sql
│   ├── 00004_create_transactions.sql
│   └── 00005_create_watchlists_and_audit_log.sql
│
└── docs/
    ├── decisions/                      # 11 ADRs covering all major choices
    ├── build-phases/                   # Written summaries of completed phases
    ├── api/                            # Generated Swagger spec
    └── architecture.md
```

---

## Testing

### Frontend

```bash
# Unit tests (Vitest)
cd frontend && ng test

# E2E tests (Playwright)
cd frontend && npx playwright test
```

Unit tests cover Angular services, form validation logic, and the `deriveHoldings()`
pure function. E2E tests cover critical user flows: register, login, create portfolio,
add transaction, verify holding appears.

### Backend

```bash
# Unit tests (no external dependencies)
cd backend && go test ./...

# Integration tests (requires Docker — uses testcontainers-go)
cd backend && go test -tags=integration ./...

# Static analysis
cd backend && go vet ./...
```

Integration tests spin up a real Postgres container via testcontainers-go and run
the full repository layer against it. No mocking at the database boundary.

Key test coverage areas:
- JWT middleware: valid token, expired token, malformed token, missing token
- Transaction service: buy increases position, sell decreases, oversell returns error
- Portfolio service: user isolation — cannot access another user's portfolios
- Repository: full CRUD with real Postgres via testcontainers

---

## Build Phases / Roadmap

| Phase | Title                         | Status      |
|-------|-------------------------------|-------------|
| 1     | Project Scaffold & Tooling    | Complete    |
| 2     | Auth Foundation               | Complete    |
| 3     | Database Schema & API         | Complete    |
| 4     | Portfolio & Transaction UI    | Complete    |
| 5     | Market Data Integration       | Not started |
| 6     | Dashboard Overview            | Not started |
| 7     | Ticker Detail View            | Not started |
| 8     | Watchlists                    | Not started |
| 9     | Connection State & Error Handling | Not started |
| 10    | Admin Dashboard               | Not started |
| 11    | Security Hardening            | Not started |
| 12    | Polish & MVP Release          | Not started |

See `docs/build-phases/` for written summaries of completed phases. See
`CLAUDE.md` for full scope, task lists, test requirements, and definition of done
for every phase.

### What Phase 5 will add

The provider abstraction (`MarketDataProvider` interface) will sit in front of
Finnhub, making it swappable. A Go WebSocket relay will proxy Finnhub's stream
to Angular clients, and an in-memory cache with TTL will reduce outbound API
calls. Angular will merge incoming price ticks into signal state using the
snapshot-plus-deltas pattern (ADR 008).

---

## Architecture Decision Records

All major technology and design choices are documented as ADRs in
`docs/decisions/`:

| ADR | Decision                                             |
|-----|------------------------------------------------------|
| 001 | Angular v21 with signals and zoneless change detection |
| 002 | Go + Gin for the API layer                          |
| 003 | Supabase for managed Postgres and Auth              |
| 004 | Finnhub as market data provider                     |
| 005 | TradingView Lightweight Charts + Chart.js combination |
| 006 | PrimeNG + Tailwind CSS component strategy           |
| 007 | Transactions as source of truth — never store holdings |
| 008 | Snapshot-plus-deltas pattern for real-time prices   |
| 009 | WCAG accessibility approach                         |
| 010 | Web-first, mobile deferred                          |
| 011 | Error handling strategy (AppError → HTTP mapping)   |

---

## Hard Rules (from CLAUDE.md)

A short list of invariants maintained throughout the codebase:

- No `any` in TypeScript — use `unknown` and narrow at the boundary
- No raw SQL string concatenation — all queries use pgx parameterized form
- No direct HTTP calls from Angular components — always through a service
- No hardcoded secrets — environment variables only
- No skipping error handling in Go — every error is checked
- No internal error details in API responses — translated at the handler layer
- **Never store derived financial values** — always compute from transactions
- Never log PII (emails, names, financial data) in plaintext
- All exported Go functions must have doc comments
- All Angular services and public methods must have TSDoc comments
