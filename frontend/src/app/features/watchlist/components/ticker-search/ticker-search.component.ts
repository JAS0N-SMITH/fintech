import {
  Component,
  DestroyRef,
  inject,
  input,
  output,
  signal,
  ChangeDetectionStrategy,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { AutoComplete } from 'primeng/autocomplete';
import { Button } from 'primeng/button';
import { InputNumber } from 'primeng/inputnumber';
import { MessageService } from 'primeng/api';
import { WatchlistService } from '../../services/watchlist.service';
import { MarketDataService } from '../../../../core/market-data.service';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import type { StockSymbol } from '../../../portfolio/models/market-data.model';

/**
 * TickerSearchComponent provides an autocomplete search for adding tickers to a watchlist.
 *
 * Users can type a symbol or select from recent/popular symbols.
 * Emits itemAdded when a symbol is successfully added, cancelled when dismissed.
 */
@Component({
  selector: 'app-ticker-search',
  standalone: true,
  imports: [FormsModule, AutoComplete, Button, InputNumber],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="space-y-4">
      <div>
        <label for="ticker-input" class="block text-sm font-medium mb-2">
          Search by ticker symbol
        </label>
        <p-autoComplete
          id="ticker-input"
          [(ngModel)]="searchInput"
          (completeMethod)="onSearch($event)"
          [suggestions]="suggestions()"
          optionLabel="symbol"
          optionValue="symbol"
          (onSelect)="selectedSymbol.set($event.value)"
          placeholder="e.g., AAPL, GOOGL"
          [minLength]="1"
          [showEmptyMessage]="true"
          emptyMessage="No symbols found"
          class="w-full"
          field="symbol"
        >
          <ng-template pTemplate="item" let-item>
            <span class="font-semibold">{{ item.symbol }}</span>
            <span class="text-sm text-surface-500 ml-2">{{ item.description }}</span>
          </ng-template>
        </p-autoComplete>
      </div>
      <div>
        <label for="target-price" class="block text-sm font-medium mb-2">
          Target price (optional)
        </label>
        <p-inputNumber
          id="target-price"
          [(ngModel)]="targetPrice"
          mode="currency"
          currency="USD"
          [minFractionDigits]="2"
          [maxFractionDigits]="2"
          placeholder="$0.00"
          class="w-full"
        />
      </div>
    </div>
    <ng-template pTemplate="footer">
      <div class="flex gap-2 justify-end">
        <p-button label="Cancel" severity="secondary" (onClick)="cancel()" />
        <p-button
          label="Add"
          [disabled]="!selectedSymbol() || isAdding()"
          [loading]="isAdding()"
          (onClick)="addItem()"
        />
      </div>
    </ng-template>
  `,
})
export class TickerSearchComponent {
  private readonly watchlistService = inject(WatchlistService);
  private readonly messages = inject(MessageService);
  private readonly marketDataService = inject(MarketDataService);
  private readonly destroyRef = inject(DestroyRef);

  readonly watchlistId = input.required<string>();
  readonly itemAdded = output<string>();
  readonly cancelled = output<void>();

  protected readonly searchInput = signal('');
  protected readonly selectedSymbol = signal<string>('');
  protected readonly targetPrice = signal<number | null>(null);
  protected readonly suggestions = signal<StockSymbol[]>([]);
  protected readonly isSearching = signal(false);
  protected readonly isAdding = signal(false);

  /**
   * Handle search input changes by querying the market data API.
   * Fetches supported symbols from the backend.
   */
  protected onSearch(event: { query: string }): void {
    const q = event.query.trim();
    if (!q) {
      this.suggestions.set([]);
      return;
    }

    this.isSearching.set(true);
    this.marketDataService
      .searchSymbols(q, 20)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (results) => {
          this.suggestions.set(results);
          this.isSearching.set(false);
        },
        error: () => {
          this.suggestions.set([]);
          this.isSearching.set(false);
          this.messages.add({
            severity: 'error',
            summary: 'Search failed',
            detail: 'Unable to search symbols. Please try again.',
          });
        },
      });
  }

  protected addItem(): void {
    const symbol = this.selectedSymbol().toUpperCase().trim();
    if (!symbol) return;

    // Validate symbol format (alphanumeric, dots, hyphens)
    if (!/^[A-Z0-9.\-]{1,20}$/.test(symbol)) {
      this.messages.add({
        severity: 'error',
        summary: 'Invalid symbol',
        detail: 'Symbol must be 1-20 characters (alphanumeric, dots, hyphens).',
      });
      return;
    }

    this.isAdding.set(true);
    this.watchlistService
      .addItem(this.watchlistId(), {
        symbol,
        target_price: this.targetPrice() ?? undefined,
      })
      .subscribe({
        next: () => {
          this.isAdding.set(false);
          this.itemAdded.emit(symbol);
        },
        error: () => {
          this.isAdding.set(false);
          this.messages.add({
            severity: 'error',
            summary: 'Add failed',
            detail: 'Could not add the ticker.',
          });
        },
      });
  }

  protected cancel(): void {
    this.cancelled.emit();
  }
}
