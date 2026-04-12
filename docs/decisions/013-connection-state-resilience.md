# ADR-013: Connection State & Error Resilience

**Status:** Accepted (Phase 9)  
**Date:** 2026-04-12  
**Context:** Phase 9 implementation

## Problem

Real-time market data applications must handle inevitable network disruptions gracefully. Users need clear visibility into connection state (connected/reconnecting/offline) and the app must automatically recover from transient failures without user intervention. Existing WebSocket reconnection logic was in place but lacked:
- User-visible connection status indicator
- Last-updated timestamps for stale data awareness
- Global error handling with user-friendly messaging
- HTTP retry logic for transient API failures (503, 429)

## Decision

Implement a four-layer resilience strategy:

### 1. WebSocket Reconnection (Transport Layer)
**Existing:** Exponential backoff in `TickerStateService.ensureWebSocket()`
- Initial delay: 1s
- Exponential multiplier: 2x per retry
- Max backoff: 30s
- Re-fetches Quote snapshots on reconnect (snapshot-plus-deltas pattern)

**New:** Connection state signal tracks three states:
- `'connected'`: WebSocket open, ticks flowing
- `'reconnecting'`: Connection lost, attempting recovery
- `'disconnected'`: No recovery attempts (permanent failure)

### 2. Last-Updated Timestamps (Data Layer)
**Implementation:** `TickerState` now includes `lastUpdated: Date | null`
- Set on snapshot fetch (REST)
- Updated on every price tick (WebSocket)
- Visible to UI via `ConnectionStatusComponent`
- Shows age of data when offline

**Rationale:** Users must know they're viewing stale data to avoid bad trading decisions.

### 3. HTTP Retry Interceptor (API Resilience)
**New:** `retryInterceptor` handles transient HTTP failures
- Retries on: 503 Service Unavailable, 429 Rate Limited
- Strategy: Exponential backoff (1s, 2s, 4s)
- Max retries: 3
- Respects `Retry-After` header for 429
- Non-retryable errors (4xx, 5xx except 503/429) fail immediately

**Placement:** Applied before auth interceptor to avoid double token refresh on retry.

### 4. Global Error Handler (User Feedback)
**New:** Angular `ErrorHandler` displays PrimeNG toasts for unhandled errors
- Generic errors: "An error occurred" (hidden details for security)
- HTTP 401: "Session expired" (redirect to login on next request)
- HTTP 429: "Too many requests, please wait" (severity: warn)
- HTTP 503: "Service temporarily unavailable" (severity: warn, retryable)
- RFC 7807 Problem Details extraction for domain-specific messages

**Rationale:** Distinguish retryable vs. permanent failures; provide actionable guidance.

### 5. Connection Status Indicator (UX)
**New Component:** `ConnectionStatusComponent` (reusable)
- Global instance in app-shell nav bar (all pages)
- Per-symbol instance on ticker detail page
- Traffic light colors: green (connected), yellow (reconnecting), red (offline)
- Shows last-updated timestamp when not connected
- Accessible (WCAG 2.1 AA): aria-labels, keyboard navigable

## Consequences

### Positive
- Users have clear visibility into data freshness and connection health
- Network blips automatically recover without user intervention
- Transient API failures retry transparently
- Stale data is clearly marked, preventing trading on outdated info
- Error messages are user-friendly and actionable
- Separation of concerns: transport, data, API, UX, feedback

### Negative
- Added complexity: 4 new files (component, error handler, interceptor, test suite)
- Exponential backoff means up to ~7s recovery time on first failure
- Retries consume extra bandwidth/API quota (mitigated by low request frequency)
- WebSocket reconnection state is app-wide, not per-symbol (acceptable for current scope)

## Trade-offs Considered

### Immediate Fail vs. Exponential Backoff
**Choice:** Exponential backoff (30s max)
- Reduces thundering herd on API outages
- Prevents rapid connection cycling (bad UX)
- User can still work offline; data resyncs on recovery

### Per-Symbol vs. App-Wide Connection State
**Choice:** App-wide (one WebSocket stream)
- Simpler architecture
- Matches current design (snapshot-plus-deltas)
- Can evolve to per-symbol in future if needed

### Toast Notifications vs. Error Page
**Choice:** Toast notifications (non-blocking)
- Doesn't interrupt workflow
- Auto-dismisses (5s)
- Severity hints urgency (warn vs. error)

## Testing

- **Unit:** TickerStateService backoff timing, connection state transitions, lastUpdated tracking
- **Unit:** RetryInterceptor 503/429 retries, backoff delays, Retry-After header parsing
- **Unit:** GlobalErrorHandler all HTTP status codes, RFC 7807 parsing, toast severity mapping
- **Unit:** ConnectionStatusComponent label/severity/accessibility for all states
- **Unit:** Go rate limiter (per-IP, per-user, 429 responses)
- **E2E:** Connection indicator visibility, offline simulation, reconnection flow

## Related ADRs
- [ADR-008: Snapshot-Plus-Deltas Pattern](008-snapshot-plus-deltas-realtime.md) — Foundation for reconnection resync
- [ADR-011: Error Handling Strategy](011-error-handling-strategy.md) — RFC 7807 Problem Details mapping

## References
- [HTTP Status Code 429: Too Many Requests](https://httpwg.org/specs/rfc6585.html#status.429)
- [HTTP Status Code 503: Service Unavailable](https://tools.ietf.org/html/rfc7231#section-6.6.4)
- [RFC 7807: Problem Details for HTTP APIs](https://tools.ietf.org/html/rfc7807)
- [WebSocket Auto-Reconnection Best Practices](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)
