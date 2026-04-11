# ADR 010: Web-First Development — Mobile Deferred

## Status

Accepted

## Context

The application could be built as a responsive web app targeting both desktop and mobile from the start, or it could target a primary platform first and expand later.

Financial portfolio management involves dense data: transaction tables, multi-column holdings views, candlestick charts, and sidebar navigation. These patterns are natural on desktop but require significant redesign for small screens — not just responsive breakpoints, but different information architecture, different navigation patterns, and different chart interactions.

Building for both simultaneously would mean either:
- Designing every component twice (desktop-first and mobile-first variants)
- Compromising the desktop layout to accommodate mobile constraints from the start

## Decision

**Build for desktop web first. Mobile is explicitly deferred — not forgotten.**

- All layout, navigation, and component design targets desktop viewports (1024px+)
- Tailwind responsive breakpoints are used where they add value without complicating the desktop layout, but mobile breakpoints are not a design requirement
- No native mobile app (React Native, Capacitor, Flutter) is planned for the current scope
- The architecture does not block mobile — a mobile-responsive layer or native app can be added later without changing the API or data model

## Consequences

**Positive:**
- Desktop layout allows the full information density the application requires: side-by-side data tables, persistent navigation, multi-panel chart views
- Faster development — one layout target instead of two
- No compromises to the desktop UX to accommodate small screens
- The Go API and Angular service layer are platform-agnostic; a mobile client could consume the same API

**Negative:**
- The application will be unusable on mobile devices until the mobile layer is built
- If mobile becomes a priority, layout work may need to be revisited rather than extended
- PrimeNG component choices should account for eventual mobile use — avoid patterns that are fundamentally non-touch-friendly

**Trigger for revisiting:**
This decision should be revisited when the MVP feature set is complete and user feedback indicates mobile demand. At that point the options are: responsive web (Tailwind breakpoints + PrimeNG responsive utilities) or a native mobile app consuming the existing API.
