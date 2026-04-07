# ADR 005: TradingView Lightweight Charts + Chart.js for Visualizations

## Status

Accepted

## Context

The dashboard requires two distinct types of charts:

1. **Financial charts** — candlestick, area, and line charts for stock price data with time-series axes, crosshairs, and real-time updates. Must handle 50K+ data points performantly.
2. **Non-financial charts** — doughnut charts for portfolio allocation, bar charts for sector breakdown, and other general-purpose visualizations.

No single charting library excels at both categories.

## Decision

Use two complementary charting libraries:

- **TradingView Lightweight Charts** (~40KB) for all financial visualizations
  - Candlestick charts on ticker detail pages
  - Area charts for portfolio performance over time
  - Volume histograms via multi-pane support
  - Real-time price tick updates without full re-renders

- **Chart.js via ng2-charts** (~67KB) for non-financial visualizations
  - Doughnut charts for portfolio allocation percentages
  - Bar charts for sector or asset class breakdown
  - Scatter plots if needed for risk/return analysis

Both libraries will be lazy-loaded using Angular's `@defer` with `on viewport` and `prefetch on idle` to minimize initial bundle impact.

## Consequences

**Positive:**
- Each library is purpose-built for its use case — better UX than a one-size-fits-all approach
- Lightweight Charts is specifically designed for financial data — handles real-time updates, crosshairs, and time-series axes natively
- Combined bundle size (~107KB) is smaller than most full-featured charting libraries
- ng2-charts provides Angular-native bindings for Chart.js with signal support
- Lazy loading means neither library impacts initial page load

**Negative:**
- Two charting APIs to learn and maintain
- Visual consistency requires careful theming to match both libraries
- Lightweight Charts has a less extensive plugin ecosystem than Chart.js

**Risks:**
- If Lightweight Charts doesn't integrate well with Angular's zoneless mode, we can wrap it in a component that manually triggers change detection via `afterRenderEffect()`
