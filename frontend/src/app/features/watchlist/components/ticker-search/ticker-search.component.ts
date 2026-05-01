import {
  Component,
  DestroyRef,
  inject,
  input,
  output,
  signal,
  ChangeDetectionStrategy,
} from '@angular/core';
import { CommonModule, DecimalPipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AutoComplete } from 'primeng/autocomplete';
import { Button } from 'primeng/button';
import { InputNumber } from 'primeng/inputnumber';
import { Textarea } from 'primeng/textarea';
import { MessageService } from 'primeng/api';
import { WatchlistService } from '../../services/watchlist.service';
import { MarketDataService } from '../../../../core/market-data.service';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import type { StockSymbol, Quote } from '../../../portfolio/models/market-data.model';

/**
 * TickerSearchComponent provides an autocomplete search for adding tickers to a watchlist.
 *
 * Users can type a symbol or select from recent/popular symbols.
 * Emits itemAdded when a symbol is successfully added, cancelled when dismissed.
 */
@Component({
  selector: 'app-ticker-search',
  standalone: true,
  imports: [CommonModule, FormsModule, AutoComplete, Button, InputNumber, Textarea, DecimalPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './ticker-search.component.html',
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
  protected readonly notes = signal('');
  protected readonly suggestions = signal<StockSymbol[]>([]);
  protected readonly isSearching = signal(false);
  protected readonly isAdding = signal(false);
  protected readonly symbolQuote = signal<Quote | null>(null);
  protected readonly selectedDescription = signal<string>('');

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

    // Check for duplicates before API call
    const alreadyAdded = this.watchlistService.items().some(i => i.symbol === symbol);
    if (alreadyAdded) {
      this.messages.add({
        severity: 'warn',
        summary: 'Already on watchlist',
        detail: `${symbol} is already in this watchlist.`,
      });
      return;
    }

    // Validate symbol format (alphanumeric, dots, hyphens)
    if (!/^[A-Z0-9.-]{1,20}$/.test(symbol)) {
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
        notes: this.notes() || undefined,
      })
      .subscribe({
        next: () => {
          this.isAdding.set(false);
          this.resetForm();
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

  protected onSymbolSelected(symbol: string): void {
    this.selectedSymbol.set(symbol);
    // Fetch quote for price preview
    this.marketDataService
      .getQuote(symbol)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (quote) => {
          this.symbolQuote.set(quote);
          // Also set description from the selected item
          const selected = this.suggestions().find(s => s.symbol === symbol);
          if (selected) {
            this.selectedDescription.set(selected.description);
          }
        },
        error: () => {
          // Silent fail - price preview is non-critical
        },
      });
  }

  protected onSearchInputChange(): void {
    // Clear selection when user types a new query
    this.selectedSymbol.set('');
    this.symbolQuote.set(null);
    this.selectedDescription.set('');
  }

  protected onClear(): void {
    this.selectedSymbol.set('');
    this.symbolQuote.set(null);
    this.selectedDescription.set('');
    this.suggestions.set([]);
  }

  private resetForm(): void {
    this.searchInput.set('');
    this.selectedSymbol.set('');
    this.targetPrice.set(null);
    this.notes.set('');
    this.symbolQuote.set(null);
    this.selectedDescription.set('');
    this.suggestions.set([]);
  }

  protected cancel(): void {
    this.cancelled.emit();
  }
}
