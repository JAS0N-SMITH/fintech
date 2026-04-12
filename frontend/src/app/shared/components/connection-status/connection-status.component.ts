import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { TagModule } from 'primeng/tag';
import { TickerStateService } from '../../../core/ticker-state.service';
import type { ConnectionState } from '../../../features/portfolio/models/market-data.model';

/**
 * ConnectionStatusComponent displays the WebSocket connection state as a colored
 * status indicator with optional last-updated timestamp information.
 *
 * Usage:
 *   <app-connection-status />                      <!-- Global indicator -->
 *   <app-connection-status symbol="AAPL" />        <!-- With stale data info -->
 */
@Component({
  selector: 'app-connection-status',
  standalone: true,
  imports: [CommonModule, TagModule],
  template: `
    <p-tag
      [value]="label()"
      [severity]="severity()"
      [attr.aria-label]="ariaLabel()"
    />
    @if (showStaleInfo() && lastUpdated()) {
      <span class="text-xs text-gray-500 ml-2">
        Last update: {{ lastUpdated() | date: 'short' }}
      </span>
    }
  `,
  styles: [`
    :host {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
    }
  `],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ConnectionStatusComponent {
  private readonly tickerStateService = inject(TickerStateService);

  /** Optional symbol for per-ticker last-updated display. If not provided, shows global status only. */
  symbol = input<string | undefined>();

  /** Computed connection state from the service. */
  connectionState = computed(() => this.tickerStateService.connectionState());

  /** Computed label based on connection state. */
  label = computed(() => {
    const state = this.connectionState();
    switch (state) {
      case 'connected':
        return 'Live';
      case 'reconnecting':
        return 'Reconnecting…';
      case 'disconnected':
        return 'Offline';
    }
  });

  /** Computed PrimeNG tag severity (color) based on state. */
  severity = computed(() => {
    const state = this.connectionState();
    switch (state) {
      case 'connected':
        return 'success';
      case 'reconnecting':
        return 'warn';
      case 'disconnected':
        return 'danger';
    }
  });

  /** Computed accessibility label. */
  ariaLabel = computed(() => {
    const sym = this.symbol();
    const state = this.connectionState();
    if (sym) {
      return `${sym} connection status: ${state}`;
    }
    return `Global connection status: ${state}`;
  });

  /** Determines if stale data info should be shown. */
  showStaleInfo = computed(() => {
    return this.connectionState() !== 'connected';
  });

  /** Computed last-updated timestamp for the symbol (if provided). */
  lastUpdated = computed(() => {
    const sym = this.symbol();
    if (!sym) return null;
    const tickers = this.tickerStateService.tickers();
    return tickers[sym]?.lastUpdated ?? null;
  });
}
