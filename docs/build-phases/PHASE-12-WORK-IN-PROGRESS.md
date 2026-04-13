# Phase 12: Polish & MVP Release — Work In Progress

**Date Started:** 2026-04-12  
**Status:** 5 of 9 sessions complete, ready for manual testing phases

---

## Completed Sessions

### ✅ Session 1: ADR & Documentation Cleanup

**Completed:**
- Renamed ADR `012-makefile-air-hot-reload.md` → `014-makefile-air-hot-reload.md` (Phase 1 tooling, not Phase 10 admin)
- Updated `CLAUDE.md` ADR index to include:
  - ADR 012: Admin dashboard architecture
  - ADR 013: Connection state & error resilience
  - ADR 014: Makefile + air hot-reload
- Fixed README.md stale content:
  - ADR table now complete (012-014)
  - Project structure section updated (dashboard/watchlist/admin descriptions)
  - API endpoints section added (watchlist, market data, admin routes)
  - Build phases table updated (all phases marked complete/in-progress)

**Files Modified:**
- `docs/decisions/014-makefile-air-hot-reload.md` (renamed from 012)
- `CLAUDE.md` (ADR index)
- `README.md` (comprehensive refresh)

---

### ✅ Session 2: Missing Build Phase Summaries

**Completed:**
- `docs/build-phases/phase-02-auth.md` — Auth foundation (Supabase, JWT, guards, interceptors)
- `docs/build-phases/phase-07-ticker-detail.md` — TradingView charts, historical bars, WebSocket streaming
- `docs/build-phases/phase-08-watchlists.md` — Watchlist CRUD, target price tracking, live prices
- Phase 12 summary (`PHASE-12-SUMMARY.md`) — to be written after all sessions complete

**Files Created:**
- 3 phase summary documents

---

### ✅ Session 3: Automated Testing — Accessibility & Visual Regression

**Completed:**
- Added `@axe-core/playwright` to frontend `package.json` devDependencies
- Updated `playwright.config.ts`:
  - Configured snapshot directory: `./e2e/snapshots/`
  - Added platform-aware snapshot paths
- Created **new E2E specs:**
  - `frontend/e2e/dashboard.spec.ts` (covers: summary cards, allocation chart, top movers, navigation, keyboard nav, accessibility)
  - `frontend/e2e/pages/dashboard.page.ts` (Page Object Model)
- **Enhanced existing E2E specs** with accessibility checks:
  - `portfolio.spec.ts` — added `checkA11y()` + visual snapshot test
  - `watchlist.spec.ts` — added `checkA11y()` + visual snapshot test
  - `ticker-detail.spec.ts` — added `checkA11y()` + visual snapshot test
  - `connection-state.spec.ts` — added `checkA11y()` + indicator screenshot
  - `admin.spec.ts` — added `checkA11y()` + visual snapshot test

**Next Steps:**
- Run `npm install` to install @axe-core/playwright
- Run `npx playwright test` to generate baseline visual regression snapshots
- Snapshots stored in `frontend/e2e/snapshots/`

---

### ✅ Session 4: Performance Budget Tests

**Completed:**
- Created `frontend/e2e/performance.spec.ts`:
  - Time-to-Interactive tests for dashboard (3s budget), portfolio list (2.5s), watchlist (2.5s)
  - Bundle size metrics logging
  - Lazy-loading verification
- Created `frontend/scripts/check-bundle-size.ts`:
  - Reads Angular build stats.json
  - Verifies bundle thresholds (main: 512KB, vendor: 500KB, default: 512KB)
  - Pretty-prints bundle analysis with percentage budgets
  - Exit code indicates pass/fail
- Updated `package.json` with `npm run check:bundle` command

**Angular Build Configuration:**
- `angular.json` already has correct budgets (verify):
  - Warning: 500 KB
  - Error: 1 MB
  - Styles warning: 4 KB, error: 8 KB

---

### ✅ Session 7: GitHub Actions CI Pipeline

**Completed:**
- Created `.github/workflows/ci.yml` with 5 jobs:

1. **Frontend Job:**
   - Setup Node.js (22.x)
   - Install & build (with budget enforcement)
   - Lint, unit tests (Vitest), npm audit, Semgrep

2. **Backend Job:**
   - Setup Go (1.26.1)
   - Build, test with race detector, coverage
   - golangci-lint, govulncheck, gosec, Semgrep

3. **E2E Job:**
   - Depends on frontend + backend passing
   - Spins up test Postgres
   - Starts backend, runs Playwright tests
   - Uploads HTML report as artifact

4. **Security Job:**
   - gitleaks for secret detection
   - detect-secrets baseline check

5. **Status Job:**
   - Summarizes all job results
   - Fails if any critical job fails

**Configuration Notes:**
- Blocks on CRITICAL/HIGH findings (gosec, govulncheck)
- Soft-fail on npm audit, semgrep (continue-on-error: true)
- E2E requires secrets: `SUPABASE_*`, `JWT_SECRET`, `FINNHUB_API_KEY`
- Reports can be inspected via Actions UI

---

## Remaining Sessions (Manual & Execution)

### ⏳ Session 5: Responsive Layout & Accessibility Manual Review

**What to do:**
1. Start dev servers:
   ```bash
   cd backend && make dev     # Terminal 1
   cd frontend && ng serve    # Terminal 2
   ```

2. Navigate to each route and verify at **multiple breakpoints** (1280, 1440, 1920px):
   - `/dashboard` — portfolio summary, charts, top movers
   - `/portfolios` — portfolio list, table responsiveness
   - `/portfolio/:id` — transactions, holdings, form fields
   - `/tickers/:symbol` — chart container, timeframe buttons
   - `/watchlist` — card grid, item table
   - `/admin/*` — user table, audit log, health metrics

3. **Keyboard Navigation:**
   - Tab through all buttons/inputs
   - Verify logical tab order
   - Verify focus indicators visible

4. **Accessibility:**
   - Check color contrast (4.5:1 for normal text, 3:1 for large)
   - Verify ARIA labels on form inputs
   - Run Lighthouse manually: `lighthouse http://localhost:4200 --view`

### ⏳ Session 6: Dark/Light Theme Consistency

**What to do:**
1. Open app in light theme (toggle via sidebar)
2. Visually inspect each page:
   - Are colors using PrimeNG surface tokens or Tailwind dark: variants?
   - Look for hardcoded `text-gray-*`, `bg-white`, `bg-gray-*` that should be theme-aware
   - Check all pages look intentional in both themes

3. **Where to check:**
   - `frontend/src/app/features/**/*.html` — component templates
   - `frontend/src/app/shared/**/*.html` — shared components
   - Search for hardcoded Tailwind color classes

4. **Fixes** (if needed):
   - Replace `text-gray-700` with `dark:text-gray-300` or PrimeNG surface var
   - Replace `bg-white` with `bg-surface` (PrimeNG token)

### ⏳ Session 8: Lighthouse Audit & Final Metrics

**What to do:**
1. **Build production bundle:**
   ```bash
   cd frontend && ng build --configuration production
   cd frontend && npm run check:bundle    # verify size
   ```

2. **Run Lighthouse locally:**
   ```bash
   npm install -g lighthouse
   cd frontend && ng serve --configuration production    # In another terminal
   lighthouse http://localhost:4200 --view
   # Scores to record: Performance, Accessibility, Best Practices, SEO
   ```

3. **Collect test coverage:**
   ```bash
   cd backend && go test -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out | tail -1    # Overall %
   ```

4. **Record metrics in** `PHASE-12-SUMMARY.md`:
   - Lighthouse Performance score
   - Lighthouse Accessibility score (aim for ≥90)
   - Lighthouse Best Practices score
   - Lighthouse SEO score
   - Initial JS bundle size (from `check:bundle`)
   - Go test coverage %
   - Frontend unit test coverage %
   - E2E test count

### ⏳ Session 9: Final Cleanup & v1.0.0 Tag

**What to do:**
1. **Remove debug code:**
   ```bash
   # Check for stray console.log in Angular
   grep -r "console.log" frontend/src/app --include="*.ts" | grep -v spec
   # Remove any found
   ```

2. **Verify production environment:**
   - `frontend/src/environments/environment.production.ts` has `production: true`
   - API URL points to correct backend

3. **Check for uncommitted changes:**
   ```bash
   git status    # Should be clean or only PHASE-12-SUMMARY.md
   ```

4. **Commit Phase 12 summary:**
   ```bash
   git add docs/build-phases/PHASE-12-SUMMARY.md
   git commit -m "docs: Phase 12 completion summary"
   ```

5. **Tag v1.0.0:**
   ```bash
   git tag -s v1.0.0 -m "MVP release: Feature-complete fintech dashboard"
   git push origin v1.0.0
   ```

---

## Running Tests & Verification

### Unit Tests (Before Session 5)
```bash
cd frontend && ng test --watch=false   # should all pass
cd backend && make test                # should all pass
```

### E2E Tests (Before Session 8)
```bash
cd frontend && npx playwright test     # includes a11y checks
```

### Bundle Size
```bash
cd frontend && ng build && npm run check:bundle
```

### Security Scanning
```bash
cd frontend && npm audit
cd backend && govulncheck ./... && gosec ./...
cd frontend && npx semgrep --config ../.semgrep.yml
```

---

## Files Created This Phase

| File | Session | Purpose |
|------|---------|---------|
| `docs/decisions/014-makefile-air-hot-reload.md` | 1 | Renamed from 012 |
| `docs/build-phases/phase-02-auth.md` | 2 | Auth foundation summary |
| `docs/build-phases/phase-07-ticker-detail.md` | 2 | Ticker detail summary |
| `docs/build-phases/phase-08-watchlists.md` | 2 | Watchlists summary |
| `frontend/e2e/dashboard.spec.ts` | 3 | Dashboard E2E tests + a11y |
| `frontend/e2e/pages/dashboard.page.ts` | 3 | Dashboard page object |
| `frontend/e2e/performance.spec.ts` | 4 | Performance budget tests |
| `frontend/scripts/check-bundle-size.ts` | 4 | Bundle size checking CLI |
| `.github/workflows/ci.yml` | 7 | GitHub Actions CI/CD |
| `PHASE-12-WORK-IN-PROGRESS.md` | — | This document |

---

## Definition of Done for Phase 12

- [x] ADR numbering conflict resolved (012 split into 012 admin, 013 connection-state, 014 makefile)
- [x] README comprehensively updated (ADRs, project structure, API endpoints)
- [x] All missing build phase summaries written (2, 7, 8)
- [x] Accessibility tests automated (axe-core in 5 E2E specs + dashboard new)
- [x] Visual regression baseline tests added (toHaveScreenshot on all main pages)
- [x] Performance budget tests configured (TTI checks, bundle size CLI)
- [x] GitHub Actions CI pipeline created (frontend, backend, E2E, security jobs)
- [ ] Manual responsive layout review completed (Session 5)
- [ ] Manual dark/light theme consistency audit completed (Session 6)
- [ ] Lighthouse scores recorded (Session 8)
- [ ] Final metrics documented in PHASE-12-SUMMARY.md (Session 8)
- [ ] v1.0.0 tagged (Session 9)

---

## Next Steps

1. **Install dependencies:** `cd frontend && npm install`
2. **Generate baseline snapshots:** `cd frontend && npx playwright test --update-snapshots`
3. **Complete Session 5:** Manual responsive layout review at breakpoints
4. **Complete Session 6:** Theme consistency audit
5. **Complete Session 8:** Lighthouse audit + metrics recording
6. **Complete Session 9:** Final tag + release

**Estimated time for remaining sessions:**
- Session 5: ~30 min (manual review)
- Session 6: ~20 min (manual review)
- Session 8: ~15 min (Lighthouse + metrics)
- Session 9: ~5 min (git tag)
