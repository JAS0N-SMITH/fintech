# Portfolio Dashboard — Build Phase Roadmap

**Pace:** 5-10 hours/week | **Milestone size:** 1-2 weeks each
**Total estimated phases:** 12 | **Estimated timeline:** 14-20 weeks to MVP

Each phase produces a working, testable increment. No phase should leave the project in a broken state. Every phase includes its own documentation deliverable so learning is captured as you go.

---

## Phase 1: Project Scaffold & Tooling (Week 1)
**Goal:** Both projects created, configured, building, and connected to source control.

**Tasks:**
- Initialize Angular v21 project with standalone bootstrap, zoneless change detection, Vitest, Tailwind CSS
- Configure PrimeNG with a base theme (Aura or Lara) and verify Tailwind + PrimeNG coexistence
- Initialize Go module with Gin, pgx, slog, and folder structure (cmd/, internal/, migrations/)
- Set up Supabase project, create connection string, verify pgx connects
- Configure ESLint (Angular) and golangci-lint (Go)
- Set up .env.example files for both projects
- Create CLAUDE.md and .claude/rules/ in the repo
- Initialize git with conventional commits, .gitignore, and branch protection
- Write initial ADRs (001 through 005 covering stack choices)

**Tests:** Angular default test passes with Vitest. Go compiles and connects to Supabase. Linting passes on both.

**Docs:** ADRs 001-005. Phase 1 build summary.

**Definition of done:** `ng serve` shows a PrimeNG component with Tailwind styling. `go run cmd/api/main.go` starts and logs a successful database ping.

---

## Phase 2: Auth Foundation (Week 2-3)
**Goal:** Users can register, log in, and access protected routes. JWT validation works end-to-end.

**Tasks:**
- Configure Supabase Auth (email/password for MVP)
- Build Angular AuthService using Supabase JS client — store access token in signal, refresh token in HTTP-only cookie
- Implement login and registration pages with Reactive Forms and PrimeNG components
- Build Go auth middleware that validates Supabase JWTs on protected endpoints
- Implement token refresh flow — silent refresh before expiration
- Create Angular auth guard (canMatch for future admin routes, canActivate for authenticated routes)
- Set up Angular HTTP interceptor to attach JWT to API requests
- Create user profiles table in Supabase (extends auth.users with role, display_name, preferences)
- Write Goose migration for profiles table

**Tests:**
- TDD: Go JWT validation middleware (valid token, expired token, malformed token, missing token)
- TDD: Go auth handler (login, register, refresh — via Supabase, but test the handler layer)
- Unit: Angular AuthService (token storage, refresh trigger, logout cleanup)
- E2E: Register → login → see protected page → logout → redirected to login

**Docs:** ADR on auth strategy (Supabase Auth + JWT validation in Go). Phase 2 build summary.

**Definition of done:** A user can register, log in, see a placeholder dashboard, log out, and be blocked from the dashboard when not authenticated.

---

## Phase 3: Database Schema & API Skeleton (Week 4)
**Goal:** Core data model exists in Supabase. Go API has CRUD endpoints for portfolios and transactions.

**Tasks:**
- Write Goose migrations for portfolios, transactions, watchlists, watchlist_items, audit_log tables
- Implement Go repository layer for portfolios (Create, GetByID, ListByUserID, Update, Delete)
- Implement Go repository layer for transactions (Create, GetByPortfolioID, Update, Delete)
- Implement Go service layer with business logic (validate transaction types, prevent negative holdings on sell)
- Implement Go handler layer with proper error handling (AppError → HTTP response mapping)
- Set up route groups: public, authenticated, admin
- Configure CORS middleware for Angular dev server origin
- Add request ID and structured logging middleware
- Generate initial Swagger docs with swaggo/swag

**Tests:**
- TDD: Transaction service — buy increases position, sell decreases, sell more than owned returns error
- TDD: Portfolio service — CRUD operations, user can only access own portfolios
- Integration: Repository tests with testcontainers-go against real Postgres
- Unit: Handler tests with httptest for request/response mapping

**Docs:** ADR on error handling strategy. API documentation via Swagger. Phase 3 build summary.

**Definition of done:** All CRUD endpoints return correct responses via Postman/curl. Integration tests pass against a real database. Swagger UI accessible at /swagger/index.html.

---

## Phase 4: Portfolio & Transaction UI (Week 5-6)
**Goal:** Users can create portfolios, log transactions, and see their holdings derived from transactions.

**Tasks:**
- Build Angular PortfolioService and TransactionService (HTTP calls to Go API)
- Build portfolio list page — PrimeNG DataTable with create/edit/delete actions
- Build transaction entry form — buy/sell/dividend with appropriate field visibility per type
- Build transaction list view — PrimeNG DataTable with sorting, filtering by portfolio and type
- Build holdings view — derived from transactions, showing symbol, quantity, average cost basis
- Wire up Angular routing: /portfolios, /portfolios/:id, /portfolios/:id/transactions
- Implement lazy loading for portfolio feature routes
- All state managed with signals — portfolio list signal, transaction list signal, computed holdings

**Tests:**
- Unit: Angular services (mock HTTP, verify request format)
- Unit: Holdings derivation logic (computed signal that aggregates transactions correctly)
- Unit: Transaction form validation (quantity required for buy/sell, not for dividend)
- E2E: Create portfolio → add buy transaction → see holding appear → add sell → quantity decreases

**Docs:** Phase 4 build summary. Document the holdings derivation logic and why it's computed, not stored.

**Definition of done:** User can manage portfolios and transactions through the UI. Holdings display correctly as derived values.

---

## Phase 5: Market Data Integration (Week 7-8)
**Goal:** Live market prices appear in the app. Provider abstraction is in place.

**Tasks:**
- Define Go MarketDataProvider interface (GetQuote, GetHistoricalBars, StreamPrices)
- Implement FinnhubProvider — REST endpoints for quotes and historical data
- Implement Go WebSocket relay — connects to Finnhub stream, pushes PriceTick to Angular clients
- Build in-memory cache with TTL for quote data in Go service layer (reduce Finnhub API calls)
- Build Angular MarketDataService — fetches quotes via REST, manages WebSocket connection
- Build Angular TickerStateService — snapshot-plus-deltas pattern, merges ticks into signal state
- Display current prices alongside holdings (current value = quantity × live price)
- Calculate and display unrealized gain/loss per position and portfolio total

**Tests:**
- TDD: FinnhubProvider (mock HTTP responses, verify normalization into Quote/PriceTick types)
- TDD: Go cache layer (TTL expiration, cache hit vs miss)
- Unit: TickerStateService merge logic (initial snapshot, tick updates price, new high/low tracked)
- Unit: Portfolio value computation (holdings × live prices)
- Integration: WebSocket relay test (connect, receive tick, verify format)

**Docs:** ADR on Finnhub selection and provider abstraction. ADR on snapshot-plus-deltas pattern. Phase 5 build summary.

**Definition of done:** Holdings show live prices. Portfolio total updates as ticks arrive. Switching away and back resyncs correctly.

---

## Phase 6: Dashboard Overview (Week 9-10)
**Goal:** The main dashboard shows portfolio summary, performance chart, and top movers at a glance.

**Tasks:**
- Build dashboard page layout — Fidelity-inspired card-based grid with left sidebar nav
- Implement app shell — sidebar navigation component with PrimeNG Menu, responsive collapse
- Integrate TradingView Lightweight Charts — portfolio performance area chart (value over time)
- Integrate Chart.js via ng2-charts — portfolio allocation doughnut chart
- Build summary cards — total portfolio value, day gain/loss, total gain/loss
- Display top movers (biggest gainers/losers in portfolio today)
- Implement dark/light theme toggle using PrimeNG theming + Tailwind dark mode
- Use @defer for chart components (lazy load on viewport, prefetch on idle)

**Tests:**
- Unit: Summary card computed signals (total value, day change, percent change)
- Unit: Chart data transformation (transactions/holdings → chart-ready data structures)
- E2E: Dashboard loads, shows portfolio summary, charts render, theme toggle works
- Visual regression: Playwright screenshot tests for dashboard layout

**Docs:** ADR on charting library combination (Lightweight Charts + Chart.js). Phase 6 build summary.

**Definition of done:** Dashboard looks and feels like a real fintech app. Charts render with real portfolio data. Theme toggle works. Sidebar navigation reaches all features.

---

## Phase 7: Ticker Detail View (Week 11)
**Goal:** Clicking a ticker shows a full detail page with price chart, key stats, and position info.

**Tasks:**
- Build ticker detail page with TradingView Lightweight Charts candlestick chart
- Implement time range selector (1D, 1W, 1M, 3M, 1Y, ALL) that fetches appropriate historical data
- Display key stats from Quote data (day range, 52-week range, volume, previous close)
- Show user's position for this ticker (quantity, cost basis, gain/loss, holding period)
- Show transaction history for this ticker filtered from portfolio transactions
- Real-time price updates via WebSocket stream on the detail page
- Volume histogram below price chart using Lightweight Charts multi-pane

**Tests:**
- Unit: Time range to API parameter mapping
- Unit: Position summary derivation for a single ticker
- E2E: Navigate to ticker → chart renders → switch time range → data updates

**Docs:** Phase 7 build summary.

**Definition of done:** Ticker detail page shows professional-quality financial charts with real-time updates and complete position context.

---

## Phase 8: Watchlists (Week 12)
**Goal:** Users can create and manage watchlists separate from their portfolios.

**Tasks:**
- Implement Go CRUD endpoints for watchlists and watchlist items
- Build Angular WatchlistService
- Build watchlist management page — create/rename/delete watchlists
- Build watchlist view — PrimeNG DataTable showing symbols with live prices, day change
- Implement ticker search/add component (autocomplete search for adding symbols)
- Optional target price field with visual indicator when current price crosses target
- Watchlist symbols subscribe to WebSocket stream for live updates

**Tests:**
- TDD: Go watchlist service (CRUD, user isolation — can't see other users' watchlists)
- Unit: Angular watchlist signals and live price integration
- E2E: Create watchlist → add ticker → see live price → remove ticker

**Docs:** Phase 8 build summary.

**Definition of done:** Watchlists are fully functional with live prices. Adding a symbol immediately starts streaming its price.

---

## Phase 9: Connection State & Error Handling (Week 13)
**Goal:** The app gracefully handles network issues, API errors, and stale data.

**Tasks:**
- Implement WebSocket reconnection with exponential backoff in Angular
- Build connection state signal (connected/reconnecting/disconnected) in TickerStateService
- Build connection status indicator component (traffic light: green/yellow/red)
- Show "last updated" timestamps on stale data when disconnected
- Implement global error handling — Angular ErrorHandler + toast notifications via PrimeNG
- Implement Go rate limiting middleware (per-user for authenticated, per-IP for public)
- Build Angular HTTP interceptor for retry logic on transient failures (503, 429 with backoff)
- Handle Finnhub API rate limit errors gracefully in Go provider

**Tests:**
- Unit: Reconnection logic (backoff timing, max retries, state transitions)
- Unit: Connection state signal updates correctly
- E2E: Simulate offline → verify indicator changes → reconnect → verify data resyncs
- TDD: Go rate limiter (allows under limit, returns 429 over limit)

**Docs:** ADR on connection state management and stale data handling. Phase 9 build summary.

**Definition of done:** App handles network disruptions without crashing. User sees clear feedback about connection state. Rate limiting protects the API.

---

## Phase 10: Admin Dashboard (Week 14-15)
**Goal:** Admin users can manage users, view audit logs, and monitor system health.

**Tasks:**
- Implement Go RBAC middleware (RequireRole) and audit logging middleware
- Build Angular admin routes with canMatch guard (prevents code download for non-admins)
- Build admin layout with separate sidebar navigation
- User management page — list users, view details, change roles, lock/unlock accounts
- Audit log viewer — PrimeNG DataTable with search, date filtering, export to CSV
- System health page — database connection status, Finnhub API status, WebSocket connection count
- Go admin endpoints: GET /admin/users, PATCH /admin/users/:id/role, GET /admin/audit-log

**Tests:**
- TDD: RBAC middleware (admin access allowed, user access returns 403)
- TDD: Audit log service (events recorded correctly, PII masked)
- Unit: Admin guard prevents non-admin navigation and code loading
- E2E: Login as admin → access admin panel → change user role → verify audit log entry
- E2E: Login as regular user → verify admin routes are inaccessible

**Docs:** ADR on admin architecture. Phase 10 build summary.

**Definition of done:** Admin can manage users and view complete audit trail. Non-admin users cannot access or even download admin code.

---

## Phase 11: Security Hardening (Week 16-17)
**Goal:** All security tooling integrated, headers configured, and security tests comprehensive.

**Tasks:**
- Configure security headers in Go (CSP with nonces, HSTS, X-Frame-Options, etc.)
- Enable Angular autoCsp in angular.json
- Set up pre-commit hooks — gitleaks for secret detection
- Integrate Semgrep with OWASP rulesets for Go and TypeScript
- Integrate gosec for Go-specific security analysis
- Configure npm audit and govulncheck in CI pipeline (or local scripts)
- Implement Supabase RLS policies as third authorization layer
- Review and harden all input validation across Go handlers
- Penetration test critical flows: auth, transaction creation, admin access
- Document all security measures and remaining risks

**Tests:**
- Security: SQL injection attempts against all endpoints
- Security: XSS payload testing via transaction notes and portfolio names
- Security: CSRF validation
- Security: JWT manipulation (altered claims, wrong signature, expired)
- Security: Rate limit enforcement under load
- Full E2E security suite

**Docs:** Security audit document. Updated threat model. Phase 11 build summary.

**Definition of done:** All security tooling runs green. No critical or high findings. Security test suite passes. Headers verified via security scanner.

---

## Phase 12: Polish & MVP Release (Week 18-20)
**Goal:** The app is polished, documented, and ready to show.

**Tasks:**
- Responsive layout review — verify all pages work at common desktop breakpoints
- Accessibility audit — keyboard navigation, screen reader testing, color contrast verification
- Performance audit — Lighthouse score, bundle size analysis, lazy loading verification
- Fix any visual inconsistencies between dark and light themes
- Write comprehensive README with setup instructions, architecture overview, screenshots
- Complete all outstanding ADRs
- Write final build phase summary covering the full project
- Record key metrics: test coverage, bundle size, Lighthouse scores
- Clean up technical debt backlog
- Tag v1.0.0

**Tests:**
- Full E2E regression suite
- Visual regression suite via Playwright
- Accessibility automated checks (axe-core)
- Performance budget tests (bundle size thresholds)

**Docs:** Complete README. Final architecture document. All ADRs indexed. Build phase summaries complete.

**Definition of done:** The app is something you'd be proud to demo in an interview. Documentation tells the full story of why decisions were made. Test suite is green and comprehensive.

---

## Post-MVP Backlog (not scheduled)

Prioritize based on interest and learning goals:

- **Tax lot tracking** — holding period calculation, short-term vs long-term capital gains indicator
- **Dividend reinvestment linking** — related_transaction_id, DRIP tracking
- **Locale support** — currency formatting, date formats, number separators per user preference
- **Mobile responsive / PWA** — responsive breakpoints, service worker, offline support
- **Options and crypto support** — extended transaction types, new asset class modeling
- ~~**Import from brokerage** — CSV import for transaction history from Fidelity, SoFi, etc.~~
- ~~**Notifications** — price alerts from watchlist target prices, portfolio threshold alerts~~
- **Materialized views** — performance optimization if needed at scale
- **Containerization** — Docker, docker-compose, container scanning (evaluate Grype or Docker Scout)
- **CI/CD pipeline** — GitHub Actions for build, test, lint, security scan, deploy
