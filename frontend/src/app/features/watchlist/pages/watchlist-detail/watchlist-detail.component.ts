import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { Button } from 'primeng/button';
import { TableModule } from 'primeng/table';
import { ConfirmationService, MessageService } from 'primeng/api';
import { ConfirmDialog } from 'primeng/confirmdialog';
import { Tag } from 'primeng/tag';
import { Dialog } from 'primeng/dialog';
import { TooltipModule } from 'primeng/tooltip';
import { WatchlistService } from '../../services/watchlist.service';
import { TickerStateService } from '../../../../core/ticker-state.service';
import { TickerSearchComponent } from '../../components/ticker-search/ticker-search.component';
import type { WatchlistItem } from '../../models/watchlist.model';

/**
 * WatchlistDetailComponent displays all items in a watchlist with live prices.
 *
 * Shows a DataTable with ticker symbols and live market data.
 * Allows adding/removing items and setting target prices.
 */
@Component({
  selector: 'app-watchlist-detail',
  standalone: true,
  imports: [
    CommonModule,
    Button,
    TableModule,
    ConfirmDialog,
    Tag,
    Dialog,
    TooltipModule,
    TickerSearchComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [ConfirmationService],
  templateUrl: './watchlist-detail.component.html',
})
export class WatchlistDetailComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  protected readonly watchlistService = inject(WatchlistService);
  readonly tickerStateService = inject(TickerStateService);
  private readonly messages = inject(MessageService);
  private readonly confirmation = inject(ConfirmationService);
  /** Watchlist ID from route params. */
  private readonly watchlistId = signal<string>('');

  /** Loading state. */
  protected readonly loading = computed(() => this.watchlistService.loading());

  /** Current watchlist items. */
  protected readonly items = computed(() => this.watchlistService.items());

  /** Get live ticker state for a symbol. */
  protected getTickerState(symbol: string) {
    return this.tickerStateService.tickers()[symbol] ?? null;
  }

  /** Compute if target price has been crossed. */
  protected getTargetPriceStatus(item: WatchlistItem): 'above' | 'below' | null {
    if (!item.target_price) return null;
    const ticker = this.getTickerState(item.symbol);
    if (!ticker || ticker.currentPrice === null) return null;
    return ticker.currentPrice >= item.target_price ? 'above' : 'below';
  }

  /** Dialog visibility for adding items. */
  protected readonly addItemDialogVisible = signal(false);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.watchlistId.set(id);
      // Load watchlist and items
      this.watchlistService.loadById(id).subscribe({
        error: () => {
          this.messages.add({
            severity: 'error',
            summary: 'Not found',
            detail: 'Could not load the watchlist.',
          });
          this.router.navigate(['/watchlists']);
        },
      });
    } else {
      this.router.navigate(['/watchlists']);
    }

  }

  protected openAddItemDialog(): void {
    this.addItemDialogVisible.set(true);
  }

  protected onItemAdded(symbol: string): void {
    this.addItemDialogVisible.set(false);
    this.messages.add({
      severity: 'success',
      summary: 'Added',
      detail: `${symbol} was added to the watchlist.`,
    });
  }

  protected onAddItemCancelled(): void {
    this.addItemDialogVisible.set(false);
  }

  protected confirmRemoveItem(item: WatchlistItem): void {
    this.confirmation.confirm({
      message: `Remove ${item.symbol} from this watchlist?`,
      header: 'Confirm removal',
      icon: 'pi pi-exclamation-triangle',
      acceptButtonStyleClass: 'p-button-danger',
      accept: () => this.removeItem(item),
    });
  }

  private removeItem(item: WatchlistItem): void {
    this.watchlistService.removeItem(this.watchlistId(), item.symbol).subscribe({
      next: () =>
        this.messages.add({
          severity: 'success',
          summary: 'Removed',
          detail: `${item.symbol} was removed.`,
        }),
      error: () =>
        this.messages.add({
          severity: 'error',
          summary: 'Remove failed',
          detail: 'Could not remove the item.',
        }),
    });
  }

  protected goBack(): void {
    this.router.navigate(['/watchlists']);
  }
}
