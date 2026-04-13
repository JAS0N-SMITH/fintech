import { Injectable, computed, effect, inject } from '@angular/core';
import { MessageService } from 'primeng/api';
import { UserPreferencesService } from '../user-preferences.service';
import { TickerStateService } from '../ticker-state.service';
import { TransactionService } from '../../features/portfolio/services/transaction.service';
import type { AlertEvent } from './alert.model';

/**
 * PortfolioAlertService monitors portfolio-level and position-level thresholds
 * configured by the user in their preferences. Alerts fire when thresholds
 * are crossed (e.g., portfolio is down more than 5%, a position is up more than 3%).
 */
@Injectable({ providedIn: 'root' })
export class PortfolioAlertService {
  private readonly preferencesService = inject(UserPreferencesService);
  private readonly tickerState = inject(TickerStateService);
  private readonly transactionService = inject(TransactionService);
  private readonly messageService = inject(MessageService);

  /** Keyed on threshold.id — tracks which thresholds have already fired. */
  private readonly _firedThresholds = new Map<string, boolean>();

  /**
   * Compute the daily portfolio change percentage.
   * Uses previous_close from TickerState (previous day's close) vs current price,
   * weighted by position size across all holdings.
   *
   * Returns null if portfolio is empty or no prices available.
   */
  private readonly portfolioDailyChangePercent = computed<number | null>(() => {
    const holdings = this.transactionService.holdings();
    const tickers = this.tickerState.tickers();

    let totalPreviousValue = 0;
    let totalCurrentValue = 0;

    for (const holding of holdings) {
      const ticker = tickers[holding.symbol];
      if (!ticker?.previousClose || !ticker.currentPrice) continue;

      const qty = parseFloat(holding.quantity);
      totalPreviousValue += ticker.previousClose * qty;
      totalCurrentValue += ticker.currentPrice * qty;
    }

    if (totalPreviousValue === 0) return null;

    return ((totalCurrentValue - totalPreviousValue) / totalPreviousValue) * 100;
  });

  constructor() {
    // Effect evaluates all configured thresholds and fires alerts on crossing
    effect(() => {
      const thresholds = this.preferencesService.preferences().thresholds;
      const holdings = this.transactionService.holdings();

      for (const threshold of thresholds) {
        let crossed = false;
        let alertTitle = '';
        let alertDetail = '';

        if (threshold.type === 'portfolio_daily_change') {
          const dailyChange = this.portfolioDailyChangePercent();
          if (dailyChange === null) continue;

          // Check if threshold is crossed
          crossed =
            threshold.direction === 'above'
              ? dailyChange >= threshold.thresholdPercent
              : dailyChange <= threshold.thresholdPercent;

          if (crossed) {
            alertTitle = `Portfolio daily change alert`;
            alertDetail = `Portfolio is ${dailyChange > 0 ? '+' : ''}${dailyChange.toFixed(2)}% (threshold: ${threshold.thresholdPercent}%)`;
          }
        } else if (threshold.type === 'position_gain_loss') {
          // Position-level alert
          if (!threshold.symbol) continue; // Invalid threshold

          const holding = holdings.find((h) => h.symbol === threshold.symbol);
          if (!holding || holding.gainLossPercent === null) continue;

          const gainLoss = holding.gainLossPercent;

          // Check if threshold is crossed
          crossed =
            threshold.direction === 'above'
              ? gainLoss >= threshold.thresholdPercent
              : gainLoss <= threshold.thresholdPercent;

          if (crossed) {
            alertTitle = `Position alert: ${threshold.symbol}`;
            alertDetail = `${threshold.symbol} is ${gainLoss > 0 ? '+' : ''}${gainLoss.toFixed(2)}% (threshold: ${threshold.thresholdPercent}%)`;
          }
        }

        const fired = this._firedThresholds.get(threshold.id) ?? false;

        if (crossed && !fired) {
          // Threshold crossed and not yet fired — fire the alert
          this._firedThresholds.set(threshold.id, true);
          this.deliverAlert({
            type: 'portfolio',
            title: alertTitle,
            detail: alertDetail,
          });
        } else if (!crossed && fired) {
          // Threshold has moved back across — reset fired state
          this._firedThresholds.set(threshold.id, false);
        }
      }
    });
  }

  /**
   * Deliver a portfolio alert via toast.
   */
  private deliverAlert(event: AlertEvent): void {
    this.messageService.add({
      severity: 'warn',
      summary: event.title,
      detail: event.detail,
      life: 8000,
    });
  }
}
