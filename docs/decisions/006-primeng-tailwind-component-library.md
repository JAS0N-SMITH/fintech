# ADR 006: PrimeNG as Single Component Library with Tailwind CSS

## Status

Accepted

## Context

The dashboard needs a rich set of interactive UI components: data tables with sorting and filtering, dropdowns, modals, toasts, menus, and form controls. Building these from scratch is time-consuming and error-prone, especially for accessibility compliance (WCAG 2.1 AA is a hard requirement per ADR 009).

We also need a utility-first styling system for layout, spacing, and responsive breakpoints. The key constraint is that Angular Material, Ant Design, and similar libraries bring their own opinionated layout systems that conflict with a utility-first approach.

Options considered:
- **PrimeNG + Tailwind** — rich component library with its own theming system; Tailwind handles layout/spacing separately
- **Angular Material + Tailwind** — Material's CSS specificity frequently fights Tailwind; known integration friction
- **Headless UI (Radix/Ark) + Tailwind** — maximum styling freedom, but requires building all interactions (focus traps, ARIA, keyboard nav) manually
- **Tailwind only** — fastest to start, but complex components (virtualized tables, date pickers) would need to be custom-built

## Decision

Use **PrimeNG** for all interactive components and **Tailwind CSS** for layout, spacing, and responsive breakpoints. The division of responsibility is clear:

- **PrimeNG owns** component internals: data tables, dropdowns, dialogs, menus, date pickers, toasts, form controls, and all associated ARIA attributes and keyboard navigation
- **Tailwind owns** layout: grid, flexbox, spacing, responsive breakpoints, and any utility styles outside component internals
- **PrimeNG theming** controls component colors, typography, and visual tokens via its design token system — Tailwind preflight is configured to not reset PrimeNG-controlled elements

## Consequences

**Positive:**
- PrimeNG ships with full WCAG 2.1 AA support out of the box — ARIA roles, keyboard navigation, focus management handled by the library
- Large component library means fewer custom components to build and maintain
- PrimeNG's theming system allows consistent visual language across all components
- Tailwind for layout is well-understood and fast to work with
- PrimeNG's `p-table` with virtual scrolling handles large transaction lists without custom implementation

**Negative:**
- Two styling systems require discipline: developers must know which layer owns which concern
- PrimeNG's CSS specificity can leak outside components; Tailwind preflight configuration must be careful
- PrimeNG upgrades occasionally introduce breaking theme changes
- Bundle size is larger than a headless approach; mitigated by lazy loading feature routes

**Configuration note:**
Tailwind's `preflight` base reset is scoped to exclude PrimeNG-controlled elements. Without this, Tailwind resets (`button { ... }`, `input { ... }`) will strip PrimeNG component styles.
