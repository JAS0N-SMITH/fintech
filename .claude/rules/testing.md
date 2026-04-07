# Testing Rules

## Strategy

- **TDD (test first):** Authentication, financial calculations, transaction processing, data validation, RBAC enforcement
- **Test-as-you-build:** UI components, layout, navigation, chart rendering
- **Every feature must be testable.** If it can't be unit tested, add it to E2E coverage.
- **Coverage targets:** 85% overall, 95%+ for auth/financial/transaction code. Use branch coverage for financial logic.

## Angular Unit Testing (Vitest)

- Vitest is the default runner — do not use Karma, Jest, or Web Test Runner
- Test standalone components by adding them to `imports` (not `declarations`) in TestBed
- Read signal values with `()` syntax, mutate with `.set()` / `.update()`
- Always call `fixture.detectChanges()` after signal mutations before DOM assertions
- Use `provideHttpClientTesting` and `HttpTestingController` for HTTP service tests
- Test pure logic (pipes, validators, signal-based state managers) without TestBed for speed
- Mock services using manually created test doubles — inject via TestBed providers
- Test the TickerStateService merge logic thoroughly: initial snapshot, tick updates, high/low tracking, reconnection resync

## Angular E2E Testing (Playwright)

- Use Page Object Model pattern for maintainable test organization
- Test critical user flows: login, view portfolio, add transaction, view ticker detail
- Test RBAC: verify non-admin users cannot access admin routes or see admin UI elements
- Test connection state: verify reconnection indicator appears and data resyncs
- Use `toHaveScreenshot()` for visual regression on dashboard layouts
- Separate E2E test suites: smoke (fast, critical paths), full (comprehensive), security (auth/RBAC)
- Run against a seeded test database, never production data

## Go Unit Testing

- Table-driven tests for all service and handler logic
- Use `t.Run()` for subtests with descriptive names
- Test files live alongside source: `service.go` and `service_test.go` in same package
- Use `httptest.NewRecorder()` for handler unit tests
- Mock repositories and providers via interface test doubles — no mocking frameworks
- Test error paths explicitly: not found, unauthorized, validation failure, database error
- Test financial calculations with precise decimal assertions (use string comparison or epsilon)

## Go Integration Testing

- Use `testcontainers-go` with Postgres module for database integration tests
- Tag integration tests with `//go:build integration` to keep unit test runs fast
- Use `testdata/` directory for SQL fixtures and seed scripts
- Test full request lifecycle: HTTP request → handler → service → repository → database → response
- Test migration up and down paths
- Test concurrent access patterns for portfolio updates

## Security Testing

- Test JWT validation: expired tokens, malformed tokens, missing tokens, wrong signing key
- Test RBAC at every endpoint: user accessing admin routes returns 403
- Test input validation: SQL injection attempts, XSS payloads, oversized inputs
- Test rate limiting: verify 429 responses after threshold
- Test audit logging: verify security events are recorded
- Include security tests in CI pipeline — never skip them

## Test Data

- Use factory functions to generate test data, not hardcoded fixtures
- Financial test data must cover edge cases: zero quantities, negative values, fractional shares, dividends
- Keep test data realistic but never use real user data
- Seed scripts for E2E should create deterministic portfolios with known expected values
