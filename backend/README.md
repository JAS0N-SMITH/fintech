# Backend README

This directory contains the Go backend for the Stock Portfolio Dashboard.

## CLI tools

The repository contains small CLI entrypoints under `cmd/` used for maintenance:

- `cmd/migrate` — run database migrations (Go/Goose)
- `cmd/seed` — reseed test data for development

Both CLIs accept flags to control `redactlog` behavior for CLI output. These flags may also be provided via environment variables parsed by the application's config.

Available flags (same for `migrate` and `seed`):

- `--redact_enabled` (default: `true`) — enable redactlog for CLI output
- `--redact_request_body` (default: `false`) — capture and redact request bodies
- `--redact_response_body` (default: `false`) — capture and redact response bodies
- `--redact_query_params` (CSV) — comma-separated sensitive query params (e.g. `access_token,api_key`)
- `--redact_header_denylist` (CSV) — comma-separated headers to denylist from logs
- `--redact_paths` (CSV) — comma-separated Pino-style redact paths to apply

Examples

Run migrations with redaction disabled:

```bash
cd backend
go run ./cmd/migrate --redact_enabled=false
```

Run the seed tool with request-body capture enabled (staging/testing only):

```bash
cd backend
go run ./cmd/seed --redact_request_body=true
```

Notes

- These flags are intended to help during debugging and in CI — capturing request/response bodies in production may expose sensitive data unless you are confident your redact rules cover all cases.
- For environment-based configuration, see the top-level `README.md` and `internal/config/config.go` for variable names (`REDACT_ENABLED`, `REDACT_REQUEST_BODY`, `REDACT_RESPONSE_BODY`, `REDACT_SENSITIVE_QUERY_PARAMS`, `REDACT_HEADER_DENYLIST`, `REDACT_PATHS`).
