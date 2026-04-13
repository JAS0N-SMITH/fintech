import { Injectable, computed, effect, inject } from '@angular/core';
import { MessageService } from 'primeng/api';
import { WatchlistService } from '../../features/watchlist/services/watchlist.service';
import { TickerStateService } from '../ticker-state.service';
import type { AlertRule, AlertEvent, AlertDirection } from './alert.model';
import type { WatchlistItem } from '../../features/watchlist/models/watchlist.model';

/**
 * PriceAlertService monitors watchlist items with target prices and fires alerts
 * when live prices cross the thresholds. Alerts fire once per crossing with
 * automatic reset when price recrosses back.
 */
@Injectable({ providedIn: 'root' })
export class PriceAlertService {
  private readonly watchlistService = inject(WatchlistService);
  private readonly tickerState = inject(TickerStateService);
  private readonly messageService = inject(MessageService);

  /** Keyed on watchlistItemId — holds mutable state for alert rules. */
  private readonly _ruleState = new Map<string, AlertRule>();

  private _notificationPermission: NotificationPermission = 'default';

  /**
   * Derive alert rules from watchlist items that have target_price set.
   * Each watchlist item with target_price becomes one rule.
   * Direction is inferred: if current price < target, direction = 'above' (expects rise).
   * If current price >= target, direction = 'below' (expects fall).
   */
  private readonly alertRules = computed<AlertRule[]>(() => {
    const items = this.watchlistService.items();
    return items
      .filter((item): item is WatchlistItem & { target_price: number } => item.target_price != null)
      .map((item) => {
        const existing = this._ruleState.get(item.id);
        const ticker = this.tickerState.tickers()[item.symbol];
        const currentPrice = ticker?.currentPrice ?? null;

        // Infer direction based on current price relative to target
        const direction: AlertDirection =
          currentPrice === null || currentPrice < item.target_price ? 'above' : 'below';

        // If a rule already exists, preserve it; otherwise create new
        if (existing) {
          existing.targetPrice = item.target_price;
          existing.direction = direction;
          return existing;
        }

        const newRule: AlertRule = {
          watchlistItemId: item.id,
          symbol: item.symbol,
          targetPrice: item.target_price,
          direction,
          fired: false,
          lastKnownPrice: currentPrice,
        };
        this._ruleState.set(item.id, newRule);
        return newRule;
      });
  });

  constructor() {
    // Effect runs after each change detection cycle, evaluating all rules
    // against the current ticker state and firing alerts on crossing detection.
    effect(() => {
      const tickers = this.tickerState.tickers();
      const rules = this.alertRules();

      for (const rule of rules) {
        const state = this._ruleState.get(rule.watchlistItemId);
        if (!state) continue;

        const tickerState = tickers[rule.symbol];
        if (!tickerState?.currentPrice) continue;

        const price = tickerState.currentPrice;

        // Detect crossing: has the price crossed the target threshold?
        const crossed =
          rule.direction === 'above'
            ? price >= rule.targetPrice
            : price <= rule.targetPrice;

        if (crossed && !state.fired) {
          // Threshold crossed and not yet fired — fire the alert
          state.fired = true;
          this.deliverAlert({
            type: 'price',
            symbol: rule.symbol,
            title: `Price alert: ${rule.symbol}`,
            detail: `${rule.symbol} hit $${price.toFixed(2)} (target: $${rule.targetPrice.toFixed(2)})`,
          });
        } else if (!crossed && state.fired) {
          // Price has moved back across threshold — reset fired state
          // so the alert can fire again when price recrosses
          state.fired = false;
        }

        state.lastKnownPrice = price;
      }
    });
  }

  /**
   * Request browser notification permission from the user.
   * Must be called from a user gesture (e.g., button click).
   */
  async requestNotificationPermission(): Promise<void> {
    if (!('Notification' in window)) return;
    this._notificationPermission = await Notification.requestPermission();
  }

  /**
   * Internal helper for tests to set notification permission without async flow.
   */
  setNotificationPermission(permission: NotificationPermission): void {
    this._notificationPermission = permission;
  }

  /**
   * Deliver an alert via toast + optionally browser notification.
   * Toast is always shown; browser notification only when tab is hidden
   * and permission has been granted.
   */
  private deliverAlert(event: AlertEvent): void {
    // PrimeNG toast — always visible
    this.messageService.add({
      severity: 'warn',
      summary: event.title,
      detail: event.detail,
      life: 8000,
    });

    // Browser Notification — only when tab is hidden and permission granted
    if ('Notification' in window && document.visibilityState === 'hidden' && this._notificationPermission === 'granted') {
      new Notification(event.title, {
        body: event.detail,
        icon: '/favicon.ico',
      });
    }
  }
}
