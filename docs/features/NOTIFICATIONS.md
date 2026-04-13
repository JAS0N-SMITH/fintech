# Notifications Feature

## Overview

The Notifications feature enables users to receive real-time alerts when watchlist prices cross target thresholds and when portfolio/position gain/loss metrics exceed configured limits. Alerts are delivered via PrimeNG toasts and browser native notifications.

**Scope:** Client-side alert engine with browser Notification API support. Server-side push notifications can be added later without architectural changes.

**Status:** Post-MVP implementation complete. Ready for E2E testing and integration with dashboard UI.

---

## User Features

### 1. Watchlist Price Alerts

Users can set target prices on watchlist items. When the live price crosses the threshold (either above or below), an alert fires once per crossing with automatic reset for re-triggers.

**How to use:**
1. Open a watchlist
2. Add a new ticker, or edit an existing item
3. Enter an optional target price (e.g., $150 for AAPL)
4. Save the watchlist item
5. When the live price crosses the target, a toast alert appears
6. If the tab is backgrounded and browser notifications are enabled, a native notification appears

**Example alerts:**
- "Price alert: AAPL — hit $151.50 (target: $150.00)" — fires when price rises above target
- "Price alert: MSFT — hit $299.75 (target: $300.00)" — fires when price falls below target

### 2. Portfolio Threshold Alerts

Users can configure alerts for portfolio-wide or per-position metrics.

**How to use:**
1. Navigate to Dashboard → Alert Settings (component available for integration)
2. Click "Add Threshold"
3. Choose alert type:
   - **Portfolio Daily Change:** Alert when daily portfolio change % exceeds threshold
   - **Position Gain/Loss:** Alert when a specific holding's gain/loss % exceeds threshold
4. For position alerts, select the symbol (e.g., AAPL)
5. Enter threshold % (negative for loss, positive for gain)
6. Save threshold
7. Alert fires when metric crosses threshold

**Example thresholds:**
- Portfolio daily change: Alert when down more than 5% (threshold: -5%, direction: Below)
- Position gain/loss: Alert when AAPL is up more than 3% (threshold: 3%, direction: Above)
- Position gain/loss: Alert when MSFT is down more than 5% (threshold: -5%, direction: Below)

### 3. Browser Notifications

When enabled, alerts fire as native browser notifications when the app tab is backgrounded.

**How to use:**
1. In Alert Settings, click "Enable Browser Notifications"
2. Grant permission when the browser prompts
3. Alerts will now appear as native notifications even if the tab is not focused

---

## Architecture

### Frontend Services

#### `PriceAlertService`
Monitors watchlist items with `target_price` set and fires alerts when live prices cross thresholds.

**Key responsibilities:**
- Derives `AlertRule[]` from `WatchlistService.items()` filtered by `target_price != null`
- Runs an Angular `effect()` on every `TickerStateService.tickers()` update
- Detects threshold crossings with one-shot-per-crossing logic via `fired` gate
- Auto-resets `fired` state when price recrosses back (allows re-trigger)
- Delivers alerts via `MessageService.add()` (toast) and `Notification` API (browser)

**Injection:** `providedIn: 'root'` — eagerly constructed via `APP_INITIALIZER`

**Key file:** `frontend/src/app/core/alerts/price-alert.service.ts`

---

#### `PortfolioAlertService`
Monitors portfolio-level and position-level thresholds configured in user preferences.

**Key responsibilities:**
- Computes `portfolioDailyChangePercent` from `previousClose` vs `currentPrice` across holdings
- Evaluates user-configured thresholds against computed metrics via `effect()`
- Uses same crossing gate pattern as `PriceAlertService`
- Supports two threshold types:
  - `portfolio_daily_change` — aggregate portfolio daily change %
  - `position_gain_loss` — individual holding gain/loss % from cost basis

**Injection:** `providedIn: 'root'` — eagerly constructed via `APP_INITIALIZER`

**Key file:** `frontend/src/app/core/alerts/portfolio-alert.service.ts`

---

#### `UserPreferencesService`
Abstracts loading and persisting user preferences (including alert thresholds) to the backend.

**Key responsibilities:**
- `load()` — fetches preferences from `GET /api/v1/me`, extracts `alert_thresholds`, updates signal
- `saveThresholds(thresholds)` — persists thresholds via `PATCH /api/v1/me/preferences`
- Type-guards API responses (converts `unknown` → `AlertPreferences`)
- Signal-based state: `preferences()` always contains current threshold list

**Injection:** `providedIn: 'root'` — lazily constructed (fetched on app init via `APP_INITIALIZER`)

**Key file:** `frontend/src/app/core/user-preferences.service.ts`

---

#### `AlertSettingsComponent`
Standalone UI component for configuring alert preferences.

**Features:**
- "Enable Browser Notifications" button (must be clicked by user, per browser policy)
- Table listing all configured thresholds with delete buttons
- Dialog to add new thresholds (type/symbol/threshold %/direction selection)
- Integrates with `UserPreferencesService` for persistence

**Styling:** PrimeNG components + Tailwind CSS

**Key file:** `frontend/src/app/features/dashboard/components/alert-settings/alert-settings.component.ts`

---

### Backend API

#### Profile Endpoints

**`GET /api/v1/me`**
Returns the authenticated user's profile including preferences.

```bash
curl -H "Authorization: Bearer $TOKEN" https://localhost:8080/api/v1/me
```

**Response:**
```json
{
  "id": "user-uuid",
  "display_name": "John Doe",
  "role": "user",
  "preferences": {
    "alert_thresholds": [
      {
        "id": "threshold-1712973600000",
        "type": "portfolio_daily_change",
        "thresholdPercent": -5.0,
        "direction": "below",
        "fired": false
      },
      {
        "id": "threshold-1712973605000",
        "type": "position_gain_loss",
        "symbol": "AAPL",
        "thresholdPercent": 3.0,
        "direction": "above",
        "fired": false
      }
    ]
  },
  "created_at": "2026-04-01T...",
  "updated_at": "2026-04-13T..."
}
```

---

**`PATCH /api/v1/me/preferences`**
Updates user preferences using JSONB merge (preserves other preference keys).

```bash
curl -X PATCH -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "alert_thresholds": [
      {
        "id": "threshold-1712973600000",
        "type": "portfolio_daily_change",
        "thresholdPercent": -5.0,
        "direction": "below",
        "fired": false
      }
    ]
  }' \
  https://localhost:8080/api/v1/me/preferences
```

**Response:** `204 No Content` on success

---

### Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ APP STARTUP                                                     │
├─────────────────────────────────────────────────────────────────┤
│ 1. APP_INITIALIZER runs                                         │
│    - Constructs PriceAlertService (registers effect)            │
│    - Constructs PortfolioAlertService (registers effect)        │
│    - Calls UserPreferencesService.load() → GET /api/v1/me       │
│    - Sets preferences signal with alert_thresholds              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ RUNTIME: WATCHLIST ITEM ADDED                                   │
├─────────────────────────────────────────────────────────────────┤
│ 1. User adds AAPL with target_price=$150 to watchlist           │
│ 2. WatchlistService.addItem() calls API, updates items signal   │
│ 3. PriceAlertService.alertRules computed re-derives             │
│    - Filter items with target_price != null                     │
│    - Create AlertRule for AAPL (targetPrice=150, direction...)  │
│ 4. TickerStateService.subscribe('AAPL') fetches live price      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ RUNTIME: PRICE TICK RECEIVED                                    │
├─────────────────────────────────────────────────────────────────┤
│ 1. WebSocket receives PriceTick { symbol: 'AAPL', price: 151 }  │
│ 2. TickerStateService.applyTick() updates tickers signal        │
│ 3. PriceAlertService effect() runs (after change detection)     │
│    - Reads alertRules() and tickers()                           │
│    - Checks: 151 >= 150 (target crossed)                        │
│    - Not yet fired → deliverAlert()                             │
│    - Set fired=true, prevents re-fire on next tick 152, 153...  │
│ 4. deliverAlert()                                               │
│    - MessageService.add() → PrimeNG toast visible               │
│    - If document.visibilityState='hidden' && permission granted  │
│      → new Notification(...)  → native OS notification          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ RUNTIME: PRICE RECROSSES (reset)                                │
├─────────────────────────────────────────────────────────────────┤
│ 1. Price drops: 151 → 149 (back below target 150)               │
│ 2. PriceAlertService effect() runs                              │
│    - Checks: 149 < 150 (not crossed) && fired=true              │
│    - Reset fired=false (allows alert to fire again if recrosses)│
│ 3. Price rises again: 149 → 152                                 │
│ 4. Alert fires again (one-shot per crossing preserved)          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ RUNTIME: PORTFOLIO THRESHOLD CONFIGURED                         │
├─────────────────────────────────────────────────────────────────┤
│ 1. User clicks "Add Threshold" in Alert Settings                │
│ 2. Configures: portfolio_daily_change, threshold=-5%, dir=below │
│ 3. Component calls UserPreferencesService.saveThresholds()      │
│ 4. PATCH /api/v1/me/preferences with alert_thresholds payload   │
│ 5. UserPreferencesService updates preferences signal            │
│ 6. PortfolioAlertService effect() re-evaluates on next tick     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Implementation Files

### Frontend

| File | Purpose |
|------|---------|
| `frontend/src/app/core/alerts/alert.model.ts` | Alert type definitions (AlertRule, PortfolioAlertThreshold, AlertPreferences, etc.) |
| `frontend/src/app/core/alerts/price-alert.service.ts` | Watchlist target price crossing engine |
| `frontend/src/app/core/alerts/price-alert.service.spec.ts` | Unit tests for PriceAlertService (11 tests) |
| `frontend/src/app/core/alerts/portfolio-alert.service.ts` | Portfolio/position threshold engine |
| `frontend/src/app/core/alerts/portfolio-alert.service.spec.ts` | Unit tests for PortfolioAlertService (8 tests) |
| `frontend/src/app/core/user-preferences.service.ts` | Preferences API abstraction |
| `frontend/src/app/features/dashboard/components/alert-settings/alert-settings.component.ts` | Alert configuration UI |
| `frontend/src/app/features/watchlist/components/ticker-search/ticker-search.component.ts` | Modified: added target price input field |
| `frontend/src/app/app.config.ts` | Modified: added APP_INITIALIZER for alert services |

### Backend

| File | Purpose |
|------|---------|
| `backend/internal/model/profile.go` | UserProfile and UpdatePreferencesInput types |
| `backend/internal/repository/profile.go` | ProfileRepository interface and pgx implementation |
| `backend/internal/service/profile.go` | ProfileService layer (minimal, delegates to repo) |
| `backend/internal/handler/profile.go` | HTTP handlers for GET /me and PATCH /me/preferences |
| `backend/cmd/api/main.go` | Modified: wired ProfileRepository → ProfileService → ProfileHandler |

---

## Testing

### Unit Tests (Frontend)

**PriceAlertService tests** (11 tests covering):
- Price rises above target (direction: above)
- Price falls below target (direction: below)
- No re-fire on subsequent ticks above/below target
- Reset and re-fire on price recross
- Null price handling
- Multiple symbols independently
- Browser notification delivery (tab hidden vs visible)
- Watchlist reload preserves fired state

**PortfolioAlertService tests** (8 tests covering):
- Portfolio daily change alert when loss exceeds threshold
- Position gain/loss alert when threshold exceeded
- Multiple thresholds evaluated independently
- Empty holdings handling
- Reset and re-fire on threshold recross

**Run tests:**
```bash
cd frontend
npx vitest run src/app/core/alerts/*.spec.ts
```

**Note:** Full `ng test` suite has pre-existing test framework migration issues (Karma/Jasmine syntax in some files) unrelated to alert implementation. Alert services follow Vitest patterns correctly.

---

### Integration Tests (Backend)

Profile handler tests should cover:
- Unauthorized access returns 401
- GET /me returns user's current preferences
- PATCH /me/preferences merges (doesn't overwrite other keys)
- Invalid JSON input returns 400

**To add integration tests:**
```go
// backend/internal/handler/profile_test.go
func TestProfileHandler_GetMe(t *testing.T) {
  // Use testcontainers-go + seeded test database
  // POST /auth/login, GET /me, verify preferences returned
}

func TestProfileHandler_UpdatePreferences(t *testing.T) {
  // PATCH /me/preferences with alert_thresholds
  // Verify JSONB merge preserves other keys
  // Re-fetch and confirm threshold persisted
}
```

---

### E2E Tests (Browser)

```typescript
// frontend/e2e/alerts.spec.ts
test('watchlist price alert fires when target is crossed', async () => {
  // 1. Add watchlist item with target_price = $150
  // 2. Mock WebSocket tick: price 149 -> 151
  // 3. Assert toast appears with "Price alert: AAPL..."
  // 4. Assert price tag shows "Above target"
});

test('portfolio threshold alert fires when daily loss exceeds limit', async () => {
  // 1. Configure threshold: portfolio_daily_change = -5%
  // 2. Mock tickers: portfolio down 6% from previous close
  // 3. Assert toast appears with portfolio alert message
});

test('browser notification fires when tab backgrounded', async () => {
  // 1. Grant notification permission via page.evaluate()
  // 2. Set visibilityState to 'hidden'
  // 3. Trigger price cross
  // 4. Assert Notification constructor called
});
```

---

## Code Patterns

### Angular Signals + Effects

```typescript
// PriceAlertService: compute rules, then detect crossings
private readonly alertRules = computed<AlertRule[]>(() => {
  // Re-derives when watchlist items or prices change
  return this.watchlistService.items()
    .filter(item => item.target_price != null)
    .map(item => ({...}));
});

constructor() {
  effect(() => {
    const rules = this.alertRules();  // Depends on watchlist items
    const tickers = this.tickerState.tickers();  // Depends on live prices
    
    for (const rule of rules) {
      // Detect crossing and fire alert
      const crossed = (rule.direction === 'above')
        ? price >= rule.targetPrice
        : price <= rule.targetPrice;
      
      if (crossed && !state.fired) {
        state.fired = true;
        this.deliverAlert({...});  // Imperative, not reactive
      }
    }
  });
}
```

### Type-Safe API Response Narrowing

```typescript
function isAlertPreferences(value: unknown): value is AlertPreferences {
  if (typeof value !== 'object' || value === null) return false;
  const obj = value as Record<string, unknown>;
  return Array.isArray(obj['thresholds']);
}

load(): Observable<AlertPreferences> {
  return this.http.get<UserProfile>(baseUrl).pipe(
    tap((profile) => {
      const alertThresholds = profile.preferences['alert_thresholds'];
      if (isAlertPreferences({ thresholds: alertThresholds })) {
        this._preferences.set({ thresholds: alertThresholds });
      }
    }),
    map(() => this._preferences()),
  );
}
```

### One-Shot Gate Pattern

```typescript
// Prevent alert spam while allowing re-trigger on recross
if (crossed && !state.fired) {
  // Threshold just crossed → fire alert once
  state.fired = true;
  this.deliverAlert({...});
} else if (!crossed && state.fired) {
  // Threshold moved back → prepare for re-fire
  state.fired = false;
}
```

---

## Future Enhancements

### Phase 1 (Easy)
- [ ] Add target_price inline edit in watchlist detail table
- [ ] Show alert history / dismissal UI
- [ ] Alert sound/vibration preferences
- [ ] Alert scheduling (quiet hours, e.g., 10pm-7am)

### Phase 2 (Medium)
- [ ] Backend push notifications (SSE or WebSocket events)
- [ ] Email alerts (on threshold cross)
- [ ] SMS alerts (Twilio integration)
- [ ] Alert groups (bundle multiple alerts if fired in quick succession)

### Phase 3 (Advanced)
- [ ] Machine learning: auto-suggest threshold values based on portfolio behavior
- [ ] Alert templates: pre-configured threshold scenarios (e.g., "Conservative," "Aggressive")
- [ ] Cross-device alerts: share alerts across mobile/desktop via background workers
- [ ] Webhook integration: POST alerts to external systems (Slack, Discord, etc.)

---

## Known Limitations

1. **Browser Notification Permission:** Must be granted per-browser per-domain. Firefox and Safari may handle permissions differently than Chrome.

2. **WebSocket Reconnection Alerts:** When the app reconnects after losing the WebSocket connection, `TickerStateService` re-fetches Quote snapshots, which may trigger alerts on the initial tick if the snapshot is already above/below target. This is expected and correct behavior (price has moved while disconnected).

3. **Portfolio Daily Change Calculation:** Uses `previousClose` from ticker state. For intraday alerts, this is accurate. For overnight gaps, the alert reflects the true daily change from market open.

4. **No Alert Persistence:** Alert history is not stored. Reloading the page resets the `fired` state, which means an alert that already fired will fire again if the price is still above/below target after reload. This is acceptable for MVP; persistence can be added to `audit_log` table in Phase 2.

5. **No Duplicate Alert Suppression:** If two watchlist items have the same symbol with different target prices (e.g., AAPL $150 and AAPL $160), both will fire independently. This is correct behavior.

---

## Troubleshooting

### "Alert not firing when price crosses target"
- Check that watchlist item has `target_price` set (not null/undefined)
- Verify TickerStateService is subscribed to the symbol (check browser dev tools → Network → WebSocket)
- Ensure PriceAlertService was injected (check if it was constructed via APP_INITIALIZER)
- Check browser console for JS errors in effect()

### "Browser notification not appearing"
- Verify notification permission was granted (browser settings)
- Check that app tab is actually backgrounded (set document.visibilityState to 'hidden')
- Ensure browser supports Notification API (not available in private/incognito mode on some browsers)
- Check if browser has notifications disabled globally

### "Portfolio alert firing unexpectedly"
- Verify threshold and direction are correct (negative = loss, positive = gain)
- Check portfolio value calculation (should use live prices from tickers)
- Confirm portfolio transactions are loaded (TransactionService.loadByPortfolio called)

---

## References

- [ADR-008: Snapshot-Plus-Deltas Pattern](../decisions/008-snapshot-plus-deltas.md)
- [ADR-013: Connection State & Error Resilience](../decisions/013-connection-state-resilience.md)
- [Angular Signals Documentation](https://angular.io/guide/signals)
- [PrimeNG MessageService](https://primeng.org/toast)
- [Browser Notification API](https://developer.mozilla.org/en-US/docs/Web/API/notification)
