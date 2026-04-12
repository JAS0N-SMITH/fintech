import { Component, input, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CardModule } from 'primeng/card';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import type { Transaction } from '../../../portfolio/models/transaction.model';

/**
 * TickerTransactionsTableComponent displays all transactions for a ticker.
 * Sorted by transaction date descending. Uses PrimeNG Table for consistent styling.
 *
 * Pure presentational component — no state or side effects.
 */
@Component({
  selector: 'app-ticker-transactions-table',
  standalone: true,
  imports: [CommonModule, CardModule, TableModule, TagModule],
  templateUrl: './ticker-transactions-table.component.html',
  styleUrls: ['./ticker-transactions-table.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TickerTransactionsTableComponent {
  readonly transactions = input.required<Transaction[]>();

  /**
   * Get the severity for the transaction type tag.
   */
  getTransactionTypeSeverity(
    type: 'buy' | 'sell' | 'dividend' | 'reinvested_dividend'
  ): 'success' | 'danger' | 'warn' | 'info' {
    switch (type) {
      case 'buy':
        return 'success';
      case 'sell':
        return 'danger';
      case 'dividend':
      case 'reinvested_dividend':
        return 'warn';
      default:
        return 'info';
    }
  }

  /**
   * Get the display label for the transaction type.
   */
  getTransactionTypeLabel(
    type: 'buy' | 'sell' | 'dividend' | 'reinvested_dividend'
  ): string {
    const labels: Record<
      'buy' | 'sell' | 'dividend' | 'reinvested_dividend',
      string
    > = {
      buy: 'Buy',
      sell: 'Sell',
      dividend: 'Dividend',
      reinvested_dividend: 'Reinvested Dividend',
    };
    return labels[type];
  }
}
