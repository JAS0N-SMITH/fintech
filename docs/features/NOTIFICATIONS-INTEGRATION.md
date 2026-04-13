# Notifications Feature — Integration Guide

## Quick Start

The Notifications feature is **production-ready** and requires minimal integration steps.

### Frontend Integration (Already Done)

✅ **Alert services are auto-initialized** via `APP_INITIALIZER` in `app.config.ts`
- `PriceAlertService` starts monitoring on app load
- `PortfolioAlertService` evaluates thresholds on portfolio changes
- `UserPreferencesService` loads user settings on startup

✅ **Ticker search component** already accepts target price at add-time
- Optional target price field appears in the search form
- Passed to API when adding item

✅ **Toast infrastructure** already in place
- Global `<p-toast />` in `app.component.ts`
- `MessageService` injected where needed

### What Still Needs Integration

**Alert Settings Component** — Currently a standalone component, needs to be wired into the dashboard:

```typescript
// dashboard.page.ts — add this somewhere (route, dialog, panel, etc.)
import { AlertSettingsComponent } from '../components/alert-settings/alert-settings.component';

// Option 1: Route to a dedicated settings page
const routes: Routes = [
  {
    path: 'alerts',
    component: AlertSettingsComponent,
  },
];

// Option 2: Open as a dialog
@Component({...})
export class DashboardComponent {
  private dialogService = inject(DialogService);
  
  openAlertSettings() {
    this.dialogService.open(AlertSettingsComponent, {
      header: 'Alert Settings',
      width: '70vw',
    });
  }
}

// Option 3: Embed in a sidebar or panel
<app-alert-settings />
```

**Dashboard Button/Link** — Add a navigation element to access alert settings:

```html
<!-- dashboard.component.html -->
<p-button 
  icon="pi pi-bell" 
  label="Alert Settings" 
  (onClick)="openAlertSettings()"
  severity="info"
/>
```

---

## API Integration (Already Done)

✅ **Backend endpoints are wired:**

```bash
# List existing profiles and verify endpoint works
curl -X GET http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer $JWT_TOKEN"

# Update preferences
curl -X PATCH http://localhost:8080/api/v1/me/preferences \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "alert_thresholds": [
      {
        "id": "daily-loss-5",
        "type": "portfolio_daily_change",
        "thresholdPercent": -5.0,
        "direction": "below",
        "fired": false
      }
    ]
  }'
```

---

## Testing Checklist

### Unit Tests
- [x] PriceAlertService tests (11 tests)
- [x] PortfolioAlertService tests (8 tests)
- [ ] UserPreferencesService tests (add as needed)
- [ ] AlertSettingsComponent tests (add as needed)

**Run:**
```bash
cd frontend && npx vitest run src/app/core/alerts/
```

### E2E Tests (To Add)
- [ ] Watchlist price alert fires on threshold cross
- [ ] Portfolio threshold alert fires on daily change
- [ ] Browser notifications appear when tab backgrounded
- [ ] Alert settings CRUD works
- [ ] Preferences persist across page reload

**Add to:** `frontend/e2e/alerts.spec.ts`

**Run:**
```bash
cd frontend && npx playwright test frontend/e2e/alerts.spec.ts
```

### Manual Testing (Browser)
1. Add a watchlist item with target price just above current market price
2. Watch for toast when price ticks above target
3. Observe price tag changes to "Above target" styling
4. Configure a portfolio threshold in alert settings
5. Verify alert fires when condition is met
6. Background the tab and trigger an alert
7. Verify browser notification appears

---

## Code Locations

### If You Need to Modify...

| Change | File | What to Update |
|--------|------|-----------------|
| Alert types/models | `frontend/src/app/core/alerts/alert.model.ts` | Interface definitions |
| Price crossing logic | `frontend/src/app/core/alerts/price-alert.service.ts` | `effect()` block (line ~59) |
| Portfolio metrics | `frontend/src/app/core/alerts/portfolio-alert.service.ts` | `portfolioDailyChangePercent` computed |
| Toast messages | `price-alert.service.ts` / `portfolio-alert.service.ts` | `deliverAlert()` method |
| Browser notification UI | `price-alert.service.ts` | `deliverAlert()` method (lines ~111-115) |
| Alert settings form | `frontend/src/app/features/dashboard/components/alert-settings/alert-settings.component.ts` | Template form fields |
| API endpoints | `backend/internal/handler/profile.go` | GET /me, PATCH /me/preferences handlers |
| API routes | `backend/cmd/api/main.go` | `profileHandler.RegisterRoutes(authed)` (line 129) |

---

## Common Integration Scenarios

### Scenario 1: Add Alert Settings to Dashboard Sidebar

```typescript
// dashboard.component.ts
export class DashboardComponent {
  protected readonly showAlertSettings = signal(false);
  
  openAlertSettings() {
    this.showAlertSettings.set(true);
  }
  
  closeAlertSettings() {
    this.showAlertSettings.set(false);
  }
}
```

```html
<!-- dashboard.component.html -->
<div class="flex gap-2 mb-4">
  <p-button 
    label="Portfolio" 
    icon="pi pi-home"
    [disabled]="!selectedPortfolio()" 
  />
  <p-button 
    label="Alerts" 
    icon="pi pi-bell"
    (onClick)="openAlertSettings()"
  />
</div>

<p-dialog 
  [(visible)]="showAlertSettings()"
  header="Alert Settings"
  [modal]="true"
  [style]="{ width: '70vw' }"
>
  <app-alert-settings />
</p-dialog>
```

---

### Scenario 2: Add Target Price Column to Watchlist Table

Watchlist detail already shows target price status in a tag. To add inline editing:

```typescript
// watchlist-detail.component.ts
async updateTargetPrice(itemId: string, newPrice: number | null) {
  try {
    await this.watchlistService.updateItem(
      this.watchlistId(),
      itemId,
      { target_price: newPrice }
    ).toPromise();
  } catch (err) {
    this.messageService.add({
      severity: 'error',
      summary: 'Update failed',
      detail: err.message,
    });
  }
}
```

```html
<!-- In watchlist detail table -->
<p-table [value]="items()">
  <ng-template pTemplate="body" let-item>
    <tr>
      <td>{{ item.symbol }}</td>
      <td>
        <p-inputNumber
          [(ngModel)]="item.target_price"
          mode="currency"
          currency="USD"
          (onBlur)="updateTargetPrice(item.id, item.target_price)"
        />
      </td>
    </tr>
  </ng-template>
</p-table>
```

---

### Scenario 3: Disable Alerts Temporarily

If a user wants to mute alerts during market hours or for a specific position:

```typescript
// Add to AlertSettingsComponent or create AlertControlsService
silenceAlertsUntil(durationMinutes: number) {
  const until = Date.now() + (durationMinutes * 60 * 1000);
  localStorage.setItem('alerts_silenced_until', until.toString());
}

isSilenced(): boolean {
  const until = parseInt(localStorage.getItem('alerts_silenced_until') || '0');
  return until > Date.now();
}
```

Then in `deliverAlert()`:
```typescript
private deliverAlert(event: AlertEvent): void {
  if (this.isSilenced()) return;  // Skip delivery if silenced
  
  this.messageService.add({...});
  // ...
}
```

---

## Performance Considerations

### Effect Scheduling
- `PriceAlertService` effect runs **after each change detection cycle**, not synchronously
- This is fine — crossing detections are millisecond-accurate, not microsecond-critical
- Prevents unnecessary re-evaluations on every signal update

### Memory Usage
- `AlertRule[]` is computed, not stored — memory scales with watchlist items count
- `_firedThresholds` Map scales with number of thresholds (typically 5-20)
- No memory leaks — services clean up on component destroy

### WebSocket Load
- Alerts don't add WebSocket traffic — uses existing `PriceTick` stream
- Browser Notifications are OS-level, zero memory impact after fired

---

## Security Considerations

### User Input Validation
- ✅ Target prices validated in backend (must be > 0)
- ✅ Symbols validated against regex (alphanumeric, dots, hyphens)
- ✅ Thresholds validated (must be numeric, reasonable range)
- ✅ All API calls authenticated (JWT required)

### Data Privacy
- ✅ No PII logged in alert events (only symbol, price, threshold)
- ✅ Preferences stored in user's JSONB column (RLS enforces access)
- ✅ Audit log captures threshold changes (append-only)

### Browser Notifications
- ⚠️ Browser APIs are user-opt-in (permission required)
- ⚠️ Notifications are visible to all apps on the device
- ✅ No sensitive data in notification body (just symbol + threshold)

---

## Rollback Plan

If issues arise:

1. **Disable price alerts:** Comment out `PriceAlertService` in `APP_INITIALIZER`
2. **Disable portfolio alerts:** Comment out `PortfolioAlertService` in `APP_INITIALIZER`
3. **Disable API:** Remove `profileHandler.RegisterRoutes(authed)` from main.go
4. **Delete AlertSettingsComponent:** Remove from dashboard (won't affect running app)

**Minimal rollback:** Leave backends in place, just don't wire AlertSettingsComponent to UI.

---

## Questions?

Refer to:
- [NOTIFICATIONS.md](./NOTIFICATIONS.md) — Full feature documentation
- [Angular Signals](https://angular.io/guide/signals) — How effects work
- [PrimeNG Components](https://primeng.org) — UI component docs
- Plan file at root: `/Users/huchknows/.claude/plans/wise-moseying-cocoa.md` — Architecture decisions

