# ADR 009: WCAG 2.1 AA Accessibility Compliance from Day One

## Status

Accepted

## Context

Financial dashboards are frequently inaccessible to users with disabilities — complex data tables, color-coded gain/loss indicators, and chart-heavy UIs are common accessibility failure points. Retrofitting accessibility into an existing application is significantly more expensive than building it in from the start.

The decision is whether to treat accessibility as a day-one requirement or defer it until the application is feature-complete.

## Decision

**WCAG 2.1 AA compliance is a non-negotiable requirement from the first component built, not a post-MVP concern.**

Specific commitments:

- **Color contrast:** minimum 4.5:1 for normal text, 3:1 for large text — enforced in PrimeNG theme configuration and any custom components
- **Keyboard navigation:** all interactive elements (tables, buttons, dropdowns, forms) must be fully keyboard navigable; PrimeNG's built-in keyboard support is used and not overridden without equivalent replacement
- **No color-only information:** gain/loss, status indicators, and alerts must use icons or text labels alongside color — a red number is not enough; it must also have an up/down arrow or label
- **Screen reader support:** all images and charts must have meaningful `alt` text or `aria-label` attributes; PrimeNG's built-in ARIA attributes are preserved
- **Chart accessibility:** TradingView Lightweight Charts and Chart.js charts must include accessible alternatives (summary tables or `aria-label` descriptions) for screen reader users who cannot perceive the visual chart
- **Testing:** VoiceOver (macOS) or NVDA (Windows) screen reader testing is part of feature development, not a QA phase

## Consequences

**Positive:**
- Accessible design patterns (clear labels, logical tab order, sufficient contrast) improve usability for all users, not just those using assistive technology
- PrimeNG's component library already implements WCAG patterns — the main cost is ensuring custom components and data visualizations follow the same standards
- Compliance from day one is significantly cheaper than retrofit
- Avoids legal risk — financial applications face increasing accessibility regulation in many jurisdictions

**Negative:**
- Chart accessibility requires additional implementation effort — providing data tables or text summaries as fallbacks for visual charts
- Color-choice constraints apply to financial gain/loss indicators (e.g., green = profit, red = loss) — these are industry conventions that we can still use, but they must not be the *only* differentiator
- Screen reader testing adds time to feature development cycles

**Implementation note:**
PrimeNG's ARIA attributes must not be overridden without providing an equivalent replacement. Any `aria-*` attribute removal requires explicit justification and an alternative approach.
