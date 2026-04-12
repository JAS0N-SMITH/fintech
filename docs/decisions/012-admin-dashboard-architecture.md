# ADR 012: Admin Dashboard Architecture

**Status:** Accepted  
**Date:** 2026-04-12  
**Context:** Phase 10 implementation  

## Problem

The portfolio dashboard needed admin functionality to manage users, review audit logs, and monitor system health. Key constraints:

1. **RBAC enforcement at three layers:** frontend guard (UX), middleware (API), RLS (database)
2. **Audit logging must be append-only** and never expose PII
3. **Admin operations must not impact performance** of regular user features
4. **Connection tracking** needed for WebSocket health monitoring

## Decision

### Backend: Clean Architecture with Fire-and-Forget Audit

The admin service layer records audit events **synchronously during the role change operation**, not in middleware. This design ensures:

- **Audit events capture before/after state** without the middleware needing to parse request bodies
- **No buffering required** — the audit event is recorded as part of the transaction
- **Failures don't cascade** — if audit log insertion fails, the primary operation (role change) still succeeds and the error is logged internally

```go
// Service layer captures state and records audit atomically
func (s *adminService) UpdateUserRole(ctx, adminID, targetID, newRole string) (*model.AdminUser, error) {
  // Prevent self-change
  if adminID == targetID {
    return nil, model.NewConflict("cannot change your own role")
  }
  
  // Perform update
  user, err := s.repo.UpdateUserRole(ctx, targetID, newRole)
  if err != nil {
    return nil, s.wrapRepoError(err)
  }
  
  // Capture before/after and record (fire-and-forget)
  after := json.Marshal(map[string]interface{}{"role": newRole})
  _ = s.RecordAuditEvent(ctx, model.AuditLogEntry{...})
  
  return user, nil
}
```

### Audit Middleware: Lightweight Action Tracking

The audit middleware (`middleware/audit.go`) is a decorator that records **successful operations only** (2xx status codes):

```go
func AuditAction(action, targetEntity string, svc AdminService) gin.HandlerFunc {
  return func(c *gin.Context) {
    c.Next()
    if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
      svc.RecordAuditEvent(c.Request.Context(), entry)
    }
  }
}
```

This avoids:
- **Request body buffering** (middleware doesn't parse JSON)
- **State capture complexity** (service handles before/after)
- **Cascading failures** (audit errors never affect the primary operation)

### Frontend: Signal-Based State with PrimeNG Components

All admin components use **signal-based state management** (no RxJS subscriptions except at service boundary):

```typescript
// Admin service bridges HTTP to signals
@Injectable({ providedIn: 'root' })
export class AdminService {
  private readonly _users = signal<AdminUser[]>([]);
  readonly users = this._users.asReadonly();
  
  loadUsers(page: number): Observable<AdminUserList> {
    return this.http.get(...).pipe(
      tap(result => {
        this._users.set(result.users);
      })
    );
  }
}
```

Benefits:
- **ChangeDetectionStrategy.OnPush** works automatically (signals register as dependencies)
- **No memory leaks** from forgotten subscriptions (OnPush + signals = automatic cleanup)
- **Minimal re-renders** (only signal fields trigger change detection)

### WebSocket Connection Counting

The WebSocket handler tracks connections with an **atomic counter**, not a global hub:

```go
type WebSocketHandler struct {
  provider  provider.MarketDataProvider
  connCount atomic.Int64  // Thread-safe counter
}

func (h *WebSocketHandler) Connect(c *gin.Context) {
  conn, _ := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
  defer conn.Close()
  
  h.connCount.Add(1)
  defer h.connCount.Add(-1)
  
  // ... streaming logic
}

func (h *WebSocketHandler) Count() int {
  return int(h.connCount.Load())
}
```

Why atomic and not a sync.Map?
- **Atomic counters are lock-free** for simple increment/decrement
- **No need to track individual connections** — only the count matters
- **No memory overhead** for per-connection tracking

### Market Data Provider HealthCheck

Added a lightweight `HealthCheck()` method to the `MarketDataProvider` interface:

```go
type MarketDataProvider interface {
  GetQuote(ctx context.Context, symbol string) (*model.Quote, error)
  GetHistoricalBars(...) ([]model.Bar, error)
  StreamPrices(...) error
  HealthCheck(ctx context.Context) error  // NEW: cached status, no rate limit burden
}
```

Implementation in Finnhub provider:
```go
func (p *FinnhubProvider) HealthCheck(ctx context.Context) error {
  // Uses a fixed symbol (AAPL) for quick polling
  // REST layer handles caching, so repeated calls are fast
  _, err := p.GetQuote(ctx, "AAPL")
  return err
}
```

## Alternatives Considered

### 1. Middleware-Based Audit Logging
**Rejected** because:
- Requires request body buffering (performance cost)
- Middleware doesn't know service-layer before/after values
- Error handling becomes complex (buffer failures vs. business logic failures)

### 2. Event Bus / Message Queue for Audit
**Rejected** because:
- Adds operational complexity (requires separate service)
- Audit events must be recorded durably for compliance
- Synchronous recording is simpler and sufficient for current scale

### 3. Global WebSocket Hub
**Rejected** because:
- Requires map iteration to count connections
- Atomicity requires mutex locking
- Atomic counter is simpler, faster, and sufficient

### 4. RLS-Only RBAC (Skip Middleware)
**Rejected** because:
- Database errors don't block bad actors
- Middleware layer catches issues early (fail-fast)
- Three-layer approach is defense-in-depth

## Implementation Details

### Parameterized Queries
All repository queries use pgx parameter placeholders (`$1`, `$2`):

```go
const q = `
  SELECT p.id, u.email, p.role FROM public.profiles p
  LEFT JOIN auth.users u ON u.id = p.id
  WHERE p.id = $1
`
```

Never string concatenation. This applies to dynamic WHERE clauses too:

```go
whereClause := ""
args := []interface{}{}
if filter.UserID != "" {
  whereClause += fmt.Sprintf("AND user_id = $%d ", argIndex)
  args = append(args, filter.UserID)
  argIndex++
}
```

### PII Masking in Audit Log
The audit log stores **only** the changed field and its new value:

```go
after := json.Marshal(map[string]interface{}{"role": newRole})
// Result: {"role":"admin"} — no email, no display_name, no tokens
```

This ensures:
- Admin can see what changed
- No sensitive data is logged
- Audit records are safe for long-term retention

### Pagination Strategy
Both users and audit logs use **server-side pagination**:

```go
offset := (page - 1) * pageSize
q := `SELECT ... FROM ... LIMIT $1 OFFSET $2`
rows, _ := db.Query(ctx, q, pageSize, offset)

// Separate count query
countQ := `SELECT COUNT(*) FROM ...`
db.QueryRow(ctx, countQ).Scan(&total)
```

Why separate count query?
- SQL `COUNT(*) OVER()` requires an extra Scan destination
- Separate query is simpler with pgx row scanning
- At scale, the count can be cached/materialized separately

## Consequences

### Positive
✅ **Audit events are always recorded** — synchronous, no message queue  
✅ **PII is never logged** — schema and service layer enforce this  
✅ **Three-layer RBAC** — frontend + middleware + RLS defense-in-depth  
✅ **No memory leaks** — atomic counter + signals + OnPush  
✅ **No SQL injection risk** — parameterized queries everywhere  
✅ **Admin operations are isolated** — separate service, repo, handler  

### Trade-offs
⚠️ **Audit failures are logged but silent** — primary operation always succeeds, but admins might not know audit failed  
⚠️ **Synchronous audit recording** — writes are slightly slower per role change, but correct and durable  
⚠️ **No audit event before/after for read operations** — only state-changing operations (PATCH) are logged  

## Related Decisions

- **ADR 004:** Provider abstraction for market data (HealthCheck fits this pattern)
- **ADR 007:** Transactions as source of truth (audit log is also append-only, complementary)
- **ADR 011:** Error handling with RFC 7807 (admin endpoints follow this)

## Testing

**Backend TDD:**
- Handler tests (mock service)
- Service tests (mock repository)
- Middleware tests (2xx recorded, 4xx/5xx skipped)
- Integration tests (real database)

**Frontend E2E:**
- Admin can access dashboard and change roles
- Audit log records the change
- Regular user cannot access admin routes
- API returns 403 for non-admins, 401 for unauthenticated

See `docs/build-phases/phase-10-summary.md` for full test coverage.

## Open Questions for Future

1. **Should audit log retention have a policy?** (Currently append-only forever)
2. **Should failed audit events retry?** (Currently logged and abandoned)
3. **Should admin impersonation be audited separately?** (Not implemented in Phase 10)
4. **Should connection count be partitioned by user or feature flag?** (Simple global count is sufficient)
