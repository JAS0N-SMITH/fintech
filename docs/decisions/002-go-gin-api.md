# ADR 002: Go with Gin Framework for API Layer

## Status

Accepted

## Context

We need a backend API server that can:

- Handle REST endpoints for CRUD operations on portfolios and transactions
- Relay real-time market data via WebSocket connections
- Validate JWTs and enforce role-based access control
- Connect to Postgres via connection pooling
- Be deployed as a single binary with minimal infrastructure

## Decision

Use Go with the Gin web framework, following clean architecture:

- **Gin** for HTTP routing and middleware (using `gin.New()` for explicit middleware control)
- **pgx** for direct Postgres access via connection pooling (`pgxpool`)
- **slog** (stdlib) for structured JSON logging
- **Viper** for configuration management (YAML files + environment variable overrides)
- **Goose** for database migrations
- **Clean architecture layers**: handlers → services → repositories/providers
- **Manual dependency injection** — no DI frameworks

## Consequences

**Positive:**
- Go compiles to a single binary — simple deployment, fast startup
- Strong concurrency primitives for WebSocket fan-out and parallel API calls
- pgx provides 30-50% better performance than ORM-based approaches
- slog is zero-dependency structured logging in the standard library
- Explicit SQL provides full control over query optimization

**Negative:**
- More boilerplate than frameworks with built-in ORM (e.g., Django, Rails)
- No built-in request validation — must use Gin's binding tags or manual validation
- Smaller web framework ecosystem compared to Node.js or Python

**Risks:**
- Clean architecture adds indirection — must ensure layers stay thin and purposeful
