import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { DecimalPipe, CurrencyPipe, CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { TableModule } from 'primeng/table';
import { Tag } from 'primeng/tag';
import type { Holding } from '../../models/transaction.model';
import type { ConnectionState } from '../../models/market-data.model';

/**
 * HoldingsTableComponent displays a read-only table of derived holdings
 * enriched with live market prices from TickerStateService.
 *
 * Holdings are computed from transactions — never stored (ADR 007).
 * Market data fields (currentPrice, gainLoss, etc.) are null until the
 * WebSocket price stream connects and delivers data.
 */
@Component({
  selector: 'app-holdings-table',
  standalone: true,
  imports: [CommonModule, RouterModule, TableModule, Tag, DecimalPipe, CurrencyPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './holdings-table.component.html',
})
export class HoldingsTableComponent {
  /** Holdings enriched with live market prices. */
  readonly holdings = input.required<Holding[]>();

  /**
   * Current WebSocket connection state.
   * Shown as a status badge so users know when prices are live.
   * Per WCAG 2.1 AA: never convey state via colour alone — icon + text label included.
   */
  readonly connectionState = input<ConnectionState>('disconnected');

  /** Returns the PrimeNG severity for the connection state badge. */
  connectionSeverity(): 'success' | 'warn' | 'danger' {
    switch (this.connectionState()) {
      case 'connected': return 'success';
      case 'reconnecting': return 'warn';
      default: return 'danger';
    }
  }

  /** Returns the human-readable label for the connection state badge. */
  connectionLabel(): string {
    switch (this.connectionState()) {
      case 'connected': return 'Live';
      case 'reconnecting': return 'Reconnecting…';
      default: return 'Offline';
    }
  }

  /** Returns the icon class for the connection state badge (WCAG: icon + text). */
  connectionIcon(): string {
    switch (this.connectionState()) {
      case 'connected': return 'pi pi-circle-fill';
      case 'reconnecting': return 'pi pi-spinner pi-spin';
      default: return 'pi pi-circle';
    }
  }

  /** Returns the Tailwind class for gain/loss colour coding. */
  gainLossClass(gainLoss: string | null): string {
    if (gainLoss === null) return 'text-surface-400';
    return parseFloat(gainLoss) >= 0 ? 'text-green-600' : 'text-red-600';
  }
}
