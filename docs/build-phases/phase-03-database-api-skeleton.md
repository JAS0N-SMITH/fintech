# Phase 3: Database Schema & API Skeleton

## Goal

Establish the full data model in Postgres and a working REST API skeleton — all CRUD endpoints return correct responses, tests pass against a real database, Swagger UI is accessible.

## What Was Built

### Database Migrations (Goose)

Five reversible SQL migrations in `backend/migrations/`:

| # | Migration | Key decisions |
|---|-----------|---------------|
| 00001 | `create_profiles` | Extends `auth.users`; `set_updated_at` and `handle_new_user` triggers use `SET search_path = ''` to satisfy Supabase security advisor |
| 00002 | `prevent_role_escalation` | DB trigger blocks any UPDATE to the `role` column, preventing users from self-escalating via the Supabase Data API |
| 00003 | `create_portfolios` | References `auth.users` directly (not profiles); RLS on all operations |
| 00004 | `create_transactions` | `CHECK` constraints enforce field presence per transaction type at the DB level (buy/sell need quantity+price; dividend needs dividend_per_share); ticker symbol regex `^[A-Z0-9.\-]{1,20}$` |
| 00005 | `create_watchlists_and_audit_log` | Audit log is append-only (no UPDATE/DELETE RLS policies) |

All tables have RLS enabled. Connections from the Go API use the service role key (bypasses RLS); Supabase client connections respect RLS.

### Error Infrastructure

Three-tier model (see ADR 011):

- **`internal/model/errors.go`** — sentinel errors + `AppError` struct with `Unwrap()` for `errors.Is`/`errors.As` chaining
- **`internal/handler/problem.go`** — `RespondError` maps `AppError` → RFC 7807 JSON; unknown errors log internally and return 500

### Domain Models

- **`internal/model/portfolio.go`** — `Portfolio`, `CreatePortfolioInput` (name required, max 100), `UpdatePortfolioInput`
- **`internal/model/transaction.go`** — `TransactionType` enum with `IsValid()`; `Transaction`; `CreateTransactionInput`; all financial values use `github.com/shopspring/decimal` (no floats)

### Repository Layer (pgx)

- **`internal/repository/portfolio.go`** — `Create`, `GetByID`, `ListByUserID` (newest first), `Update`, `Delete`; returns `model.ErrNotFound` for `pgx.ErrNoRows`
- **`internal/repository/transaction.go`** — same CRUD plus `QuantityHeld` which computes `SUM(buy+reinvested) - SUM(sell)` via a single aggregate query; used by service for sell validation

### Service Layer

- **`internal/service/portfolio.go`** — ownership enforcement on `GetByID`, `Update`, `Delete` (callerID vs `p.UserID` → `ErrForbidden`); wraps repository `ErrNotFound` in `AppError`
- **`internal/service/transaction.go`** — `validateTransactionInput` checks type-specific required fields; sell path calls `QuantityHeld` → `ErrConflict` if overselling; `assertOwnership` for all mutations

### Handler Layer

- **`internal/handler/portfolio.go`** — `List`, `Create`, `GetByID`, `Update`, `Delete`; reads `user_id` from `ContextKeyUserID` set by auth middleware
- **`internal/handler/transaction.go`** — `List`, `Create`, `Delete` nested under `/portfolios/:portfolioID/transactions`
- Routes registered on the authenticated route group in `cmd/api/main.go`

### Swagger / API Docs

- `swaggo/swag` annotations on all handler methods
- `swag init` generates `backend/docs/` (docs.go, swagger.json, swagger.yaml)
- Swagger UI served at `http://localhost:8080/swagger/index.html`
- Re-generate after handler changes: `cd backend && swag init -g cmd/api/main.go -o docs`

## Tests Written

| Layer | File | Count | Approach |
|-------|------|-------|----------|
| Handler | `handler/problem_test.go` | 6 | httptest, table-driven |
| Handler | `handler/portfolio_test.go` | 16 | httptest + mock service, stub auth middleware |
| Handler | `handler/transaction_test.go` | 15 | httptest + mock service |
| Service | `service/portfolio_test.go` | 13 | TDD, in-memory mock repo |
| Service | `service/transaction_test.go` | 22 | TDD, covers buy/sell/dividend/validation/ownership |
| Repository | `repository/portfolio_integration_test.go` | 7 | testcontainers-go, real Postgres |
| Repository | `repository/transaction_integration_test.go` | 11 | testcontainers-go, real Postgres |

Integration tests require Docker and are guarded by `//go:build integration`. Run with:

```bash
cd backend && go test -tags=integration ./internal/repository/...
```

Unit tests run without Docker:

```bash
cd backend && go test ./...
```

## Architecture Notes

- **No holdings table.** All current quantities are derived from transaction history via `QuantityHeld`. This was a deliberate decision (ADR 007) to eliminate sync bugs between stored state and transaction history.
- **`decimal.Decimal` zero value and gin binding.** `binding:"required"` does not catch a zero-value `decimal.Decimal` because it is a struct (not a pointer). `total_amount > 0` is enforced at the database `CHECK` constraint level.
- **RLS bypassed in tests.** Integration tests connect as the Postgres superuser (bypassing RLS). The test helper stubs the `auth` schema and runs full goose migrations so the schema under test is identical to production.

## Definition of Done — Checklist

- [x] All 5 migrations applied to Supabase (up and reversible)
- [x] All CRUD endpoints return correct responses
- [x] Unit tests pass (`go test ./...`)
- [x] Integration tests pass against real Postgres (`go test -tags=integration ./internal/repository/...`)
- [x] Swagger UI accessible at `/swagger/index.html`
- [x] ADR 011 written and indexed in CLAUDE.md
- [x] `go vet ./...` clean
