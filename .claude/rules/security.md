# Security Rules

## Authentication

- Supabase Auth handles login/signup on the Angular side
- Go API validates Supabase JWTs on every authenticated request via middleware
- Access tokens: 15-minute lifetime, RS256 signing
- Refresh tokens: 7-day lifetime, server-side rotation (new refresh token issued on each use, old one invalidated)
- Store access token in memory only (Angular service signal) — never localStorage
- Refresh token in HTTP-only, Secure, SameSite=Strict cookie
- JWT claims: minimal — user ID, role, expiration. No sensitive data in payload.
- On token refresh failure, redirect to login — do not retry indefinitely

## RBAC (Role-Based Access Control)

- Roles: user, admin (expandable later)
- Three enforcement layers, all required:
  1. **Angular guards** — UX convenience, prevents navigation (canMatch for admin code splitting)
  2. **Go middleware** — API enforcement, returns 403 for unauthorized access
  3. **Supabase RLS** — Database-level enforcement as defense-in-depth
- Never trust client-side role checks as a security boundary
- Admin routes use stacked middleware: Auth → RequireRole("admin") → AuditLog

## Input Validation

- Validate all input server-side in Go handlers before passing to services
- Use Gin's binding and validation tags for request struct validation
- Sanitize all string inputs — strip HTML, validate against expected patterns
- Validate ticker symbols against allowed character set (alphanumeric, dots, hyphens)
- Validate numeric financial inputs: reject negative quantities, NaN, infinity
- Maximum request body size enforced at middleware level
- Client-side validation in Angular is for UX only — never rely on it for security

## API Security

- CORS: specify exact allowed origins — never use wildcard (*) with credentials
- Rate limiting: per-user for authenticated endpoints, per-IP for public endpoints
- All responses include security headers:
  - `Content-Security-Policy` with nonces (Angular autoCsp: true)
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Permissions-Policy` restricting unused browser APIs
- Use RFC 7807 Problem Details for error responses — never expose stack traces or internal paths

## Audit Logging

- Log all security-relevant events to the audit_log table (append-only, never update/delete)
- Events to capture: login, logout, failed login, role change, transaction create/update/delete, admin actions
- Each entry: user_id, action, target_entity, target_id, before_value, after_value, ip_address, user_agent, timestamp
- Mask PII before storage in audit logs
- Admin impersonation (if implemented) must log every action taken while impersonating

## Secrets Management

- All secrets via environment variables — JWT signing keys, Supabase URL/key, Finnhub API key
- Never commit secrets to git — use .env.example as template, .env in .gitignore
- Rotate API keys periodically — design key loading to support hot-reload from environment

## Development Workflow Security Tooling

- **Pre-commit:** gitleaks or detect-secrets to prevent secret commits
- **SAST:** Semgrep (Go + TypeScript OWASP rulesets), gosec for Go-specific issues
- **Dependency auditing:** npm audit (frontend), govulncheck (backend), run in CI
- **DAST:** OWASP ZAP against staging (manual or CI)
- **Container scanning:** Deferred until containerization phase. Evaluate Grype (Anchore) or Docker Scout. Do NOT use Trivy — compromised in March 2026 supply chain attack (CVE-2026-33634), investigation still ongoing.
- CI pipeline blocks on CRITICAL or HIGH severity findings

## WebSocket Security

- Authenticate WebSocket connections with JWT before upgrading
- Validate subscription requests — users can only subscribe to symbols, not arbitrary channels
- Implement connection limits per user to prevent resource exhaustion
- Log WebSocket connect/disconnect events for monitoring
