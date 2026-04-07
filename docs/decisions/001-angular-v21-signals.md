# ADR 001: Use Angular v21 with Signals-First Architecture

## Status

Accepted

## Context

We need a frontend framework for a dashboard-heavy fintech application that will display real-time market data, portfolio summaries, and interactive charts. The framework must support:

- Reactive UI updates driven by live WebSocket data
- Strong typing for financial calculations
- Component-based architecture for complex layouts
- Mature ecosystem for data tables, forms, and charting

## Decision

Use Angular v21 with the following configuration:

- **Zoneless change detection** — opt out of zone.js, rely on signals for reactivity
- **Standalone components** — no NgModules, every component is standalone
- **Signals-first state management** — `signal()` for state, `computed()` for derived values, `effect()` for side effects
- **RxJS for streams only** — WebSocket connections, HTTP requests, and complex async orchestration
- **Vitest** as the unit test runner (Angular 21 default)

## Consequences

**Positive:**
- Signals provide fine-grained reactivity without zone.js overhead
- Zoneless mode eliminates unnecessary change detection cycles — critical for high-frequency tick updates
- Strong TypeScript integration catches financial calculation errors at compile time
- Angular's opinionated structure reduces decision fatigue for a solo project

**Negative:**
- Zoneless change detection is experimental — may encounter edge cases with third-party libraries
- Signals-first is a paradigm shift from traditional RxJS-heavy Angular patterns — fewer community examples available
- Angular's learning curve is steeper than alternatives for someone unfamiliar with the framework

**Risks:**
- If zoneless causes issues with PrimeNG, we can fall back to `provideZoneChangeDetection()` with minimal code changes
