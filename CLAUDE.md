# Portfolio Dashboard — CLAUDE.md

## Project Overview

Fintech stock portfolio dashboard for tracking brokerage account positions, watchlists, and real-time market data. This is a learning project — prioritize clear documentation, explain architectural decisions, and follow established fintech patterns.

**Stack:** Angular v21 (frontend) | Go/Gin (API) | Supabase (Postgres) | Finnhub (market data)
**Component Library:** PrimeNG (user + admin dashboards)
**Styling:** Tailwind CSS for layout/utility, PrimeNG theming for components
**Charting:** TradingView Lightweight Charts (financial) + Chart.js via ng2-charts (non-financial)
**Testing:** Vitest (Angular unit) | Playwright (E2E) | Go stdlib testing + testcontainers
**Auth:** Supabase Auth (frontend) + JWT validation in Go middleware

## Commands

```bash
# Frontend
cd frontend && ng serve                    # Dev server with hot-reload
cd frontend && ng test                     # Unit tests (Vitest)
cd frontend && npx playwright test         # E2E tests
cd frontend && ng build                    # Production build

# Backend (using Makefile)
cd backend && make dev                     # Dev server with hot-reload (requires air)
cd backend && make run                     # Dev server once
cd backend && make test                    # All unit tests
cd backend && make test-integration        # Integration tests
cd backend && make lint                    # Linting with golangci-lint
cd backend && make vet                     # Static analysis
cd backend && make help                    # Show all available commands

# Security
cd frontend && npm audit                   # Dependency audit
cd backend && govulncheck ./...            # Go vulnerability check
cd backend && gosec ./...                  # Go security scanner
```

### Backend Setup (First Time)

Install air for hot-reload development:
```bash
go install github.com/air-verse/air@latest
```

Then use `make dev` for auto-reloading server (like `ng serve` for frontend).

## Project Structure

```
project-root/
├── CLAUDE.md
├── frontend/src/app/
│   ├── core/                  # Guards, interceptors, auth service, tokens
│   ├── features/
│   │   ├── dashboard/         # Portfolio overview, summary charts
│   │   ├── portfolio/         # Holdings, transactions, position detail
│   │   ├── watchlist/         # Watchlist management
│   │   ├── ticker-detail/     # Individual stock view with charts
│   │   └── admin/             # User mgmt, audit logs, feature flags
│   ├── shared/                # Reusable components, pipes, directives
│   ├── app.component.ts
│   ├── app.config.ts
│   └── app.routes.ts
├── backend/
│   ├── cmd/api/               # Application entrypoint
│   ├── internal/
│   │   ├── handler/           # HTTP handlers (parse request, return response)
│   │   ├── service/           # Business logic layer
│   │   ├── repository/        # Database access (pgx)
│   │   ├── provider/          # External API integrations (Finnhub, etc.)
│   │   ├── middleware/        # Auth, RBAC, rate limiting, audit, CORS
│   │   ├── model/             # Domain types (Quote, PriceTick, Transaction, etc.)
│   │   └── config/            # Viper configuration
│   ├── migrations/            # Goose SQL migrations
│   └── testdata/              # SQL fixtures for integration tests
└── docs/
    ├── decisions/             # Architecture Decision Records
    ├── build-phases/          # Phase summaries
    ├── api/                   # Generated Swagger docs
    └── architecture.md        # System design overview
```

## Hard Rules

- No `any` in TypeScript — use `unknown` and narrow
- No raw SQL string concatenation — use parameterized queries via pgx
- No direct HTTP calls from Angular components — always go through services
- No hardcoded secrets — environment variables only, use .env files locally
- No skipping error handling in Go — every error must be checked
- No internal error details in API responses — translate at the handler boundary
- Never store derived financial values in the database — calculate from transactions
- Never log PII (emails, names, financial data) in plaintext
- All exported Go functions must have doc comments
- All Angular services and public methods must have TSDoc comments

## Conventions

- **Git commits:** Conventional Commits format (feat:, fix:, docs:, test:, refactor:, security:)
- **Branches:** feature/, bugfix/, security/, docs/ prefixes
- **Go errors:** Return sentinel errors from repositories, wrap with AppError in services, map to HTTP in handlers
- **Angular state:** Signals for synchronous state, RxJS only for streams (WebSocket, HTTP). Bridge with toSignal()/toObservable()
- **Testing rule:** If it involves money, auth, or data validation — write the test first (TDD). UI features — test as you build.
- **Coverage targets:** 85% overall, 95%+ for auth, financial calculations, and transaction processing

## Detailed Rules

See `.claude/rules/` for domain-specific conventions:
- `angular.md` — Signals-first patterns, standalone components, control flow, PrimeNG usage
- `go-api.md` — Clean architecture layers, error handling, middleware, pgx patterns
- `testing.md` — TDD workflow, Vitest patterns, Go table-driven tests, E2E strategy
- `security.md` — Auth flow, RBAC, input validation, security headers, audit logging
- `data-model.md` — Entity relationships, transaction-as-source-of-truth, provider abstraction

## Architecture Decision Records

See `docs/decisions/` for full records. Read relevant ADRs before modifying related code.
When creating a new ADR, also append its number and title to this index.

- 001: Use Angular v21 with signals-first architecture
- 002: Go with Gin framework for API layer
- 003: Supabase for database hosting
- 004: Finnhub as primary market data provider with provider abstraction
- 005: TradingView Lightweight Charts + Chart.js for visualizations
- 006: PrimeNG as single component library with Tailwind CSS
- 007: Transactions as single source of truth — no holdings table
- 008: Snapshot-plus-deltas pattern for real-time data
- 009: WCAG 2.1 AA accessibility compliance from day one
- 010: Web-first development — mobile deferred
- 011: Error handling strategy — sentinel errors, AppError, RFC 7807 Problem Details
- 012: Admin dashboard architecture — fire-and-forget audit, three-layer RBAC, atomic counters
