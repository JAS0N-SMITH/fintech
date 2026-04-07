# Go API Rules

## Clean Architecture Layers

- **Handlers** parse HTTP requests, call services, return HTTP responses. No business logic.
- **Services** encode business logic, orchestrate repositories and providers. Define interfaces they depend on.
- **Repositories** handle database access via pgx. Return domain models and sentinel errors.
- **Providers** wrap external APIs (Finnhub, etc.). Normalize vendor responses into app domain types.
- Dependencies flow inward: handler → service → repository/provider. Never reverse.
- Manual dependency injection — wire in cmd/api or a dedicated server/dependency.go file. No DI frameworks.

## Interfaces

- Define interfaces at the consumer, not the provider ("accept interfaces, return structs")
- Keep interfaces small — 1-3 methods preferred
- Mock via test doubles (manual structs), not mocking frameworks
- Provider interface must abstract all market data operations so providers are swappable

## Error Handling

- Every error must be checked — no discarding errors with `_`
- Repositories return sentinel errors: `ErrNotFound`, `ErrDuplicate`, `ErrConflict`
- Services wrap errors into `AppError` types with business context and appropriate error codes
- Handlers map `AppError` to HTTP responses using `errors.As()` — never expose internal details
- Use RFC 7807 Problem Details format for all error responses
- Log full error details internally with slog, return sanitized message to client

## Database (pgx + Supabase)

- Use `pgxpool` for connection pooling (25 max connections, 5 min idle, 1hr max lifetime)
- Always use parameterized queries — never string concatenation for SQL
- Use Supabase session pooler (port 5432) for persistent connections
- Goose for migrations — SQL migration files in backend/migrations/
- All migrations must be reversible (include both up and down)
- Use transactions for multi-table operations
- No ORM — write explicit SQL for clarity and performance

## Middleware Stack Order

RequestID → Logging → Recovery → CORS → RateLimit → (Auth → RBAC → AuditLog for protected routes)

- Use `gin.New()` not `gin.Default()` to control middleware explicitly
- RequestID generates and propagates correlation ID via context
- All middleware passes context.Context through layers for request tracing

## Logging

- Use `slog` (stdlib) for structured JSON logging
- Every log entry includes `request_id` from context
- Log levels: DEBUG for development detail, INFO for request lifecycle, WARN for recoverable issues, ERROR for failures
- Never log PII, tokens, passwords, or financial data in plaintext
- Log sanitized audit events for security-relevant actions

## Route Organization

```
v1 := r.Group("/api/v1")
public := v1.Group("/")           // POST /auth/login, /auth/register, /auth/refresh
authenticated := v1.Group("/")    // Auth middleware
admin := v1.Group("/admin")       // Auth + RequireRole("admin") + AuditLog
```

## Configuration

- Viper with YAML config files for non-sensitive defaults
- Environment variables override config for all sensitive values (JWT secret, DB URL, API keys)
- Never commit .env files — use .env.example as template

## Code Style

- Follow Go Code Review Comments (go.dev/wiki/CodeReviewComments)
- All exported types, functions, and methods must have godoc comments starting with the name
- Use `doc.go` for package-level documentation
- Receiver names: short, consistent, never `this` or `self`
- Group imports: stdlib, external, internal (separated by blank lines)
- Run `gofmt` and `go vet` before every commit
