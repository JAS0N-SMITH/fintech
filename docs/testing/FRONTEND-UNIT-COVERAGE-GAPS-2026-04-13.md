# Frontend Unit Coverage Gaps (2026-04-13)

## Snapshot

- Test files: 23 passed
- Tests: 252 passed
- Statements: 52.19%
- Branches: 58.13%
- Functions: 59.46%
- Lines: 55.39%

## High-Coverage Areas (Good)

- Core services are mostly strong, including auth, market data, retry interceptor, Supabase token handling, and alert services.
- Portfolio data services and shared connection status are also high confidence.

## Largest Gaps

Coverage is currently dragged down by UI-heavy component/template surfaces and chart rendering paths.

### Very low coverage component areas

- dashboard allocation chart component/html
- dashboard performance chart component/html
- ticker key stats card component/html
- ticker position summary card component/html
- ticker chart component/html
- transactions table component/html

### Moderate gaps still worth improving

- ticker detail page template paths and edge branches
- ticker search component behavior branches
- app shell template behavior and some branch logic
- import dialog template paths (component logic already high)

## Why these are low

- Many specs intentionally isolate logic with template overrides to keep tests deterministic.
- Chart/UI libraries increase runtime complexity in unit tests, so test suites focused on class behavior instead of template interaction.
- A number of defensive branches and display-only states are not exercised yet.

## Recommended next backend-facing move

Frontend unit suite is stable and green. Continue with backend unit tests and fix failures there before adding more frontend coverage tests.

## Follow-up options (frontend, later)

1. Add shallow template-interaction tests for dashboard/ticker cards.
2. Add focused branch tests for app shell and ticker search.
3. Add dedicated chart-wrapper unit tests with mocked render APIs.
4. Add one targeted coverage gate for core paths to prevent regressions while UI coverage improves gradually.
