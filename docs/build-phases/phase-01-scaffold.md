# Phase 1: Project Scaffold & Tooling — Build Summary

**Date:** 2026-04-06
**Duration:** Phase 1 of 12

## What Was Set Up

### Frontend (Angular v21)
- Angular 21.2.6 with standalone components, zoneless change detection, and Vitest
- Tailwind CSS v4 integrated via `--style tailwind` (CSS-first config, `@import "tailwindcss"`)
- PrimeNG 21.1.5 with Aura theme preset via `@primeuix/themes`
- Angular ESLint 21.3.1 with `@typescript-eslint/no-explicit-any: "error"`
- 2016 file naming convention (`app.component.ts` style)
- Demo component proving PrimeNG + Tailwind + signals work together
- Environment files for development and production configuration

### Backend (Go/Gin)
- Go 1.26.1 (darwin/arm64 native)
- Gin 1.12.0 web framework with explicit middleware control (`gin.New()`)
- pgx v5.9.1 for Postgres connection pooling
- Viper 1.21.0 for configuration (`.env` + environment variables)
- slog structured JSON logging (stdlib)
- Clean architecture folder structure: `cmd/api/`, `internal/{handler,service,repository,provider,middleware,model,config}/`
- Health check endpoint: `GET /api/v1/health`
- golangci-lint 2.11.4 with errcheck, govet, staticcheck, unused, ineffassign, revive, gosec
- Graceful startup without database — logs warning and starts server

### Documentation
- ADRs 001-005 covering Angular, Go/Gin, Supabase, Finnhub, and charting library decisions
- `.env.example` files for both frontend and backend

## Tool Versions

| Tool | Version |
|------|---------|
| Node.js | 20.19.4 |
| npm | 11.12.1 |
| Angular CLI | 21.2.6 |
| Go | 1.26.1 (arm64) |
| Gin | 1.12.0 |
| pgx | 5.9.1 |
| PrimeNG | 21.1.5 |
| Tailwind CSS | 4.x (via Angular integration) |
| Vitest | Angular default |
| golangci-lint | 2.11.4 |
| goose | 3.27.0 |

## Deviations from Plan

- `@primeng/themes` package is deprecated; migrated to `@primeuix/themes` (same Aura preset, different package name)
- Go installed as 1.26.1 instead of planned 1.23.4 (Homebrew installed latest)
- golangci-lint v2 requires formatters (gofmt, goimports) in `formatters` section, not `linters`

## Verification Results

- `ng serve` — PrimeNG button renders with Tailwind layout classes
- `ng test` — Vitest tests pass
- `ng lint` — ESLint passes with no-any enforcement
- `go build ./...` — compiles cleanly
- `go vet ./...` — passes
- `golangci-lint run` — 0 issues
- `go run cmd/api/main.go` — starts server, logs config warning (DATABASE_URL not set)
- `curl localhost:8080/api/v1/health` — returns `{"status":"ok"}`

## Next Phase

Phase 2: Auth Foundation — Supabase Auth integration, JWT validation middleware, login/registration pages.
