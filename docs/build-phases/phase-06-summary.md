# Phase 6: Dashboard Overview — Build Summary

**Dates:** Week 9-10 (April 2026)
**Status:** ✅ Complete
**Goal:** Build the main dashboard with portfolio summary, live prices, charts, and top movers. Implement the app shell with persistent sidebar navigation and dark/light theme toggle.

---

## What Was Built

### 1. App Shell & Navigation (`AppShellComponent`)
**Files:**
- `frontend/src/app/shared/layout/app-shell/app-shell.component.ts`
- `frontend/src/app/shared/layout/app-shell/app-shell.component.html`
- `frontend/src/app/shared/layout/app-shell/app-shell.component.spec.ts`

**Features:**
- Two-column layout: responsive sidebar + main content area
- Sidebar collapses to icon-only mode (16px vs 256px width)
- Navigation menu with Dashboard, Portfolios, Watchlists links
- Admin section visible only to admin users (conditional rendering)
- User profile card showing email and role
- Theme toggle button (light/dark)
- Logout button
- Integrated with PrimeNG `p-menu`, `p-button`, `p-avatar` components

**Design Notes:**
- Sidebar is collapsible via local signal `isSidebarCollapsed`
- Admin items checked at render time from `AuthService.user()` role
- Theme toggle wired to `ThemeService`
- Logout calls `AuthService.signOut()` which clears session and redirects to login

---

### 2. Theme Service (`ThemeService`)
**File:** `frontend/src/app/core/theme.service.ts` + `.spec.ts`

**Features:**
- Singleton service managing light/dark mode state
- Signal-based theme state: `isDark: Signal<boolean>`
- Persists preference to localStorage (`app-theme`)
- Side effect that updates `document.documentElement.classList` with `.dark` class
- Methods: `toggle()`, `setDark(boolean)`
- Loads initial preference from localStorage on construction

**Integration:**
- PrimeNG configured with `darkModeSelector: '.dark'` in `app.config.ts`
- Tailwind CSS respects dark mode via `@supports` block in `styles.css`
- Components read `isDark()` signal to adapt UI (icon colors, etc.)

---

### 3. Dashboard Page (`DashboardComponent`)
**Files:**
- `frontend/src/app/features/dashboard/pages/dashboard/dashboard.component.ts`
- `frontend/src/app/features/dashboard/pages/dashboard/dashboard.component.html`
- `frontend/src/app/features/dashboard/pages/dashboard/dashboard.component.spec.ts`

**Features:**

#### Summary Cards (3 metrics)
1. **Total Portfolio Value** — Sum of all holdings' current values across all portfolios
2. **Total Unrealized Gain/Loss** — Sum of position gains/losses + percentage
3. **Day Gain/Loss** — Sum of intraday changes (quantity × (currentPrice - previousClose)) + percentage

#### Allocation Chart
- Displays portfolio allocation by symbol as a doughnut chart
- Powered by new `AllocationChartComponent` (see below)
- Shows allocation percentages and table breakdown

#### Top Movers Tables
- **Top 5 Gainers** — symbols sorted by day change % (descending)
- **Top 5 Losers** — symbols sorted by day change % (ascending)
- Each row shows: symbol, current price, day change %, with color coding (green/red)

#### Data Flow
1. On init: loads all portfolios via `PortfolioService.loadAll()`
2. For each portfolio: loads transactions via `TransactionService.loadByPortfolio(id)`
3. Aggregates all transactions into `allTransactions` signal
4. Derives holdings using `deriveHoldings()` pure function (reuses portfolio logic)
5. Enriches holdings with live prices from `TickerStateService`
6. Subscribes to all symbols found in holdings

**Computed Signals:**
- `allHoldings` — derived from aggregated transactions
- `enrichedHoldings` — enriched with live prices
- `totalPortfolioValue` — sum of current values
- `totalUnrealizedGainLoss` — sum of position P&L
- `dayGainLoss` — intraday change
- `topGainers` / `topLosers` — sorted/sliced holding lists

**Testing:**
- Unit tests for computed signals (empty state, zero values)
- Mock services for portfolio/transaction/ticker services
- E2E tests will verify data loads and displays (deferred to Phase 9)

---

### 4. Allocation Chart Component (`AllocationChartComponent`)
**Files:**
- `frontend/src/app/features/dashboard/components/allocation-chart/allocation-chart.component.ts`
- `frontend/src/app/features/dashboard/components/allocation-chart/allocation-chart.component.html`
- `frontend/src/app/features/dashboard/components/allocation-chart/allocation-chart.component.css`

**Features:**
- Input: `holdings: Holding[]` (with live prices)
- Displays portfolio allocation as PrimeNG doughnut chart
- Includes allocation details table with symbol, value, and percentage
- Color-coded symbols with distinct palette (blue, green, amber, red, purple, cyan)
- Legend positioned at bottom
- Responsive height (h-64 in grid)

**Chart Data Computation:**
- Filters out zero-value holdings
- Computes allocation % = holding value / total portfolio value × 100
- Sorts by value descending

**Implementation Notes:**
- Uses `ng2-charts` `p-chart` component with `type: 'doughnut'`
- Chart data is computed signal, re-renders when holdings change
- Table provides accessibility and detailed view for screen readers

---

### 5. Dashboard Routes Update
**File:** `frontend/src/app/features/dashboard/dashboard.routes.ts`

**Changes:**
- `AppShellComponent` is now the parent component for all dashboard routes
- Children routes:
  - `''` → `DashboardComponent` (home page)
  - `'portfolios'` → loads portfolio feature routes
  - `'watchlists'` → loads watchlist routes (placeholder for Phase 8)

**Effect:**
- Sidebar navigation persists across all dashboard features
- Breadcrumb/context remains visible while navigating

---

### 6. TickerState Enhancement
**File:** `frontend/src/app/features/portfolio/models/market-data.model.ts`

**Change:**
- Added `previousClose: number | null` field to `TickerState` interface
- Populated from initial Quote snapshot in `TickerStateService.applySnapshot()`
- Enables day gain/loss calculation (current price - previous close)

**Usage:**
- Dashboard and top movers tables rely on this for intraday % change

---

### 7. Dark Mode Configuration
**Files Modified:**
- `frontend/src/styles.css` — Added `@supports` block for dark mode color scheme
- `frontend/src/app/app.config.ts` — Added `darkModeSelector: '.dark'` to PrimeNG config

**Implementation:**
- PrimeNG v4 uses CSS-based theming with class selector on `<html>`
- Tailwind v4 uses `@supports` for dark mode detection
- Theme preference persisted to localStorage via `ThemeService`

---

## Dependencies Added

```bash
npm install lightweight-charts ng2-charts chart.js
```

- **lightweight-charts** — TradingView library (installed, reserved for Phase 7)
- **ng2-charts** — Angular wrapper for Chart.js
- **chart.js** — Core charting library

---

## Architecture Decisions

### 1. **Client-Side Aggregation**
All portfolio aggregation (summing holdings across portfolios, computing totals) happens client-side via computed signals. No server-side `/portfolios/summary` endpoint needed.

**Why:** Keeps backend simple, leverages existing transaction + quote APIs, and enables real-time updates as ticks arrive.

### 2. **Snapshot-Plus-Deltas for Day Gain/Loss**
Day gain/loss computation relies on `previousClose` from initial Quote snapshot, not a running day high/low.

**Why:** `previousClose` is stable across the day; computing from ticks would require tracking baseline. ADR 008 pattern fits naturally.

### 3. **AllocationChartComponent as Reusable Input**
Allocation chart accepts `holdings` as input; not responsible for fetching or aggregating.

**Why:** Composition over inheritance. Dashboard can reuse same chart for portfolio-level or account-level views. Testable in isolation.

### 4. **Deferred Chart Components**
Portfolio performance chart deferred to Phase 7 (placeholder visible).

**Why:** Historical bar data aggregation (computing portfolio value over time) is complex and doesn't affect core MVP. Lazy loading deferred components saves bundle size.

---

## What Still Needs Work

1. **E2E Tests** (Phase 9)
   - Verify dashboard loads with real portfolio data
   - Test theme toggle persistence across page reload
   - Verify top movers sort correctly

2. **Visual Regression Tests** (Phase 12)
   - Playwright screenshot tests for dashboard layout (light/dark modes)
   - Sidebar collapse/expand visual consistency

3. **Portfolio Performance Chart** (Phase 7)
   - TradingView Lightweight Charts integration
   - Fetch 1-month historical bars for all symbols
   - Aggregate portfolio value per date (sum of symbol quantities × close price)

4. **Accessibility Audit** (Phase 12)
   - Screen reader testing: verify card labels, table headers are announced
   - Keyboard navigation: Tab through sidebar menu items
   - Color contrast: verify green/red gain/loss colors meet WCAG AA

---

## Testing Coverage

- ✅ **ThemeService** unit tests (toggle, localStorage persistence, class toggling)
- ✅ **AppShellComponent** unit tests (sidebar collapse, theme toggle, logout)
- ⚠️ **DashboardComponent** basic unit tests (computed signals, empty state)
- ⚠️ **AllocationChartComponent** — integration tests pending (Chart.js rendering)
- 📋 **E2E tests** deferred to Phase 9 (requires running backend + seeded data)

---

## Key Files

```
frontend/src/app/
├── core/
│   ├── theme.service.ts (+spec)           [New]
│   ├── ticker-state.service.ts            [Modified: added previousClose population]
├── shared/
│   └── layout/
│       └── app-shell/
│           ├── app-shell.component.ts (+spec)     [New]
│           ├── app-shell.component.html           [New]
│           └── app-shell.component.css            [New]
├── features/
│   ├── dashboard/
│   │   ├── pages/
│   │   │   └── dashboard/
│   │   │       ├── dashboard.component.ts (+spec) [New]
│   │   │       ├── dashboard.component.html       [New]
│   │   │       └── dashboard.component.css        [New]
│   │   ├── components/
│   │   │   └── allocation-chart/
│   │   │       ├── allocation-chart.component.ts  [New]
│   │   │       ├── allocation-chart.component.html [New]
│   │   │       └── allocation-chart.component.css  [New]
│   │   └── dashboard.routes.ts            [Modified: wired AppShell]
│   ├── watchlist/
│   │   └── watchlist.routes.ts            [New: placeholder for Phase 8]
│   └── portfolio/
│       └── models/
│           └── market-data.model.ts       [Modified: added previousClose to TickerState]
├── app.config.ts                          [Modified: added darkModeSelector to PrimeNG]
└── app.component.ts                       [Unchanged: still bare shell]

frontend/src/
└── styles.css                             [Modified: added dark mode color-scheme support]
```

---

## Verification Steps

```bash
# Build
ng build

# Dev server
ng serve   # localhost:4200

# Unit tests
ng test

# Manual verification
1. Login → dashboard shows with sidebar
2. Click sidebar icons → collapse/expand works smoothly
3. Click theme toggle → light/dark mode switches, persists on refresh
4. Sidebar menu: click Dashboard → home page, click Portfolios → routes to /portfolios
5. Summary cards: show numeric values if portfolios exist; zero if none
6. Allocation chart: doughnut and table display allocation percentages
7. Top movers: tables show symbols sorted by day change %
8. Navigation: sidebar remains visible when navigating between routes
```

---

## Phase 6 Complete ✅

The dashboard is now the visual center of the app. Users see:
- A persistent, collapsible sidebar with navigation and theme toggle
- A portfolio overview dashboard with key metrics
- Real-time price integration (from Phase 5)
- Allocation visualization
- Top daily movers for quick insight

Next: Phase 7 will add the ticker detail page with full candlestick charts and time range selection.
