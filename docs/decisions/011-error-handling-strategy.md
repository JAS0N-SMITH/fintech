# ADR 011: Error Handling Strategy — Sentinel Errors, AppError, RFC 7807

## Status

Accepted

## Context

The API needs a consistent, layered error handling approach that:

- Keeps each architectural layer responsible for only its own concerns
- Never leaks internal error details (stack traces, SQL messages, file paths) to clients
- Returns machine-readable error responses that Angular can parse reliably
- Supports `errors.Is` / `errors.As` for idiomatic Go error inspection through wrapped chains
- Maps business errors to the correct HTTP status codes without coupling the service layer to HTTP

## Decision

We use a three-tier model: **sentinel errors** in repositories, **AppError** in services, **RFC 7807 Problem Details** in handlers.

### Tier 1 — Repository: sentinel errors

Repositories return a small set of package-level sentinel values from `internal/model`:

```go
var (
    ErrNotFound  = errors.New("not found")
    ErrDuplicate = errors.New("duplicate")
    ErrConflict  = errors.New("conflict")
    ErrForbidden = errors.New("forbidden")
    ErrValidation = errors.New("validation")
)
```

The repository never constructs HTTP status codes or user-facing messages. It translates database-specific errors (e.g., `pgx.ErrNoRows`) to these sentinels so the layer above is decoupled from pgx.

### Tier 2 — Service: AppError

Services wrap sentinels (or create fresh ones) into `AppError`, which carries:

- **Code** — the wrapped sentinel (`ErrNotFound`, etc.)
- **Message** — a safe, user-facing string (e.g., `"portfolio not found"`)
- **HTTPStatus** — the appropriate status code

`AppError.Unwrap()` returns the sentinel, so `errors.Is(appErr, model.ErrNotFound)` works even through `errors.Join` wrapping:

```go
// services use constructors, not raw structs
return nil, model.NewNotFound("portfolio")
return nil, model.NewForbidden()
return nil, model.NewConflict("cannot sell more than held")
return nil, model.NewValidation("quantity is required for buy")
```

### Tier 3 — Handler: RFC 7807 Problem Details

`handler.RespondError` uses `errors.As` to unwrap the chain and find an `AppError`. If found, it responds with the embedded HTTP status and message. Unknown errors map to 500 and are logged internally without exposing details to the client:

```json
{
  "status": 404,
  "title": "Not Found",
  "detail": "portfolio not found"
}
```

The `Problem` struct follows [RFC 7807](https://www.rfc-editor.org/rfc/rfc7807) and is consistent across all error responses, making Angular error handling uniform.

## Consequences

**Positive:**
- Service and repository layers are fully decoupled from HTTP — they can be reused in CLI tools, background workers, or tests without a running HTTP server
- `errors.Is` / `errors.As` work through `errors.Join` wrapping, so middleware or composed errors resolve correctly
- Internal details never reach clients; the logging boundary is explicit in `RespondError`
- Angular parses a single `Problem` shape for all errors

**Negative / trade-offs:**
- Three layers to trace when debugging an error (though each layer is small and explicit)
- `AppError` embeds `HTTPStatus`, which is technically an HTTP concern living in the model package — accepted as pragmatic; the alternative (a handler-side mapping table) adds more indirection without clear benefit
- Validation messages surface to the client; care must be taken to keep them safe and not leak implementation details
