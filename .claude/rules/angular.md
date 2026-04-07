# Angular Rules

## Signals-First Architecture

- Use `signal()` for all component and service state — never BehaviorSubject for synchronous state
- Use `computed()` for derived values (portfolio totals, gain/loss, allocation percentages)
- Use `effect()` sparingly and only for side effects (logging, localStorage, chart initialization)
- Bridge RxJS streams to signals at the service boundary with `toSignal()`
- Bridge signals to observables only when needed with `toObservable()`
- RxJS is for streams only: WebSocket connections, HTTP requests, complex async orchestration

## Components

- All components are standalone — no NgModules
- Use `input()`, `output()`, and `model()` signal APIs — never decorator-based @Input/@Output
- Use `ChangeDetectionStrategy.OnPush` on every component (zoneless default reinforces this)
- Use `inject()` function for dependency injection — never constructor injection
- Destroy subscriptions and chart instances in `ngOnDestroy` or use `DestroyRef`
- Use `afterRenderEffect()` for DOM-dependent initialization (chart rendering)

## Templates

- Use `@if`, `@for`, `@switch` control flow — never *ngIf, *ngFor, *ngSwitch
- `@for` requires `track` — use a unique identifier, never track by index for data that changes
- Use `@defer` with `on viewport` and `prefetch on idle` for below-fold chart components
- Never call functions in templates that perform calculations — use `computed()` signals instead

## Routing

- Lazy load all feature routes with `loadChildren`
- Use functional guards (`canMatch`, `canActivate`, `canActivateChild`) — never class-based
- `canMatch` for admin routes to prevent code download for non-admin users
- Route-level providers for feature-scoped services

## Forms

- Use Reactive Forms for production features
- Experimental Signal Forms acceptable for new non-critical screens
- Always validate on both client (UX) and server (security) — client validation is not a security boundary

## PrimeNG

- Use PrimeNG components for all interactive elements: tables, dropdowns, dialogs, menus, toasts
- Use PrimeNG theming system for consistent colors and typography
- Configure Tailwind preflight to not reset PrimeNG-controlled elements
- Tailwind handles layout, spacing, responsive breakpoints — PrimeNG handles component internals

## Project Organization

- Group by feature domain, not technical type
- `providedIn: 'root'` for global services (auth, WebSocket, market data)
- Feature-scoped services registered via route providers
- Shared module for truly reusable components, pipes, and directives
- Avoid barrel files (index.ts) within features — use direct imports
- One component per file, filename matches component selector

## Accessibility (WCAG 2.1 AA)

- Use PrimeNG's built-in ARIA attributes — don't override without reason
- All interactive elements must be keyboard navigable
- Color contrast minimum 4.5:1 for normal text, 3:1 for large text
- Never convey information through color alone — use icons or text labels alongside
- All images and charts must have meaningful alt text or aria-labels
- Test with screen reader (VoiceOver or NVDA) during feature development
