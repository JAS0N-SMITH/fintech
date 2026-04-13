import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnDestroy,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { ViewChild } from '@angular/core';
import { CurrencyPipe, DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Button } from 'primeng/button';
import { TabsModule } from 'primeng/tabs';
import { TableModule } from 'primeng/table';
import { Dialog } from 'primeng/dialog';
import { ConfirmDialog } from 'primeng/confirmdialog';
import { Tag } from 'primeng/tag';
import { Select } from 'primeng/select';
import { PortfolioService } from '../../services/portfolio.service';
import { TransactionService } from '../../services/transaction.service';
import { TickerStateService } from '../../../../core/ticker-state.service';
import { HoldingsTableComponent } from '../../components/holdings-table/holdings-table.component';
import { TransactionFormComponent } from '../../components/transaction-form/transaction-form.component';
import { ImportDialogComponent } from '../../components/import-dialog/import-dialog.component';
import type { Transaction, TransactionType, CreateTransactionInput } from '../../models/transaction.model';

const TYPE_LABELS: Record<string, string> = {
  buy: 'Buy',
  sell: 'Sell',
  dividend: 'Dividend',
  reinvested_dividend: 'Reinvested div.',
};

const TYPE_FILTER_OPTIONS = [
  { label: 'All types', value: null },
  { label: 'Buy', value: 'buy' },
  { label: 'Sell', value: 'sell' },
  { label: 'Dividend', value: 'dividend' },
  { label: 'Reinvested dividend', value: 'reinvested_dividend' },
];

const TYPE_SEVERITY: Record<string, 'success' | 'danger' | 'info' | 'secondary'> = {
  buy: 'success',
  sell: 'danger',
  dividend: 'info',
  reinvested_dividend: 'secondary',
};

/**
 * PortfolioDetailComponent shows a portfolio's holdings and transaction history.
 *
 * Holdings are derived from transactions via TransactionService.holdings (computed signal).
 * The "Add transaction" dialog feeds into TransactionService.create, which automatically
 * updates both the transaction list and the holdings derivation.
 */
@Component({
  selector: 'app-portfolio-detail',
  standalone: true,
  imports: [
    CurrencyPipe,
    DatePipe,
    FormsModule,
    Button,
    TabsModule,
    TableModule,
    Dialog,
    ConfirmDialog,
    Tag,
    Select,
    HoldingsTableComponent,
    TransactionFormComponent,
    ImportDialogComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [ConfirmationService],
  templateUrl: './portfolio-detail.component.html',
})
export class PortfolioDetailComponent implements OnInit, OnDestroy {
  @ViewChild(ImportDialogComponent) importDialog?: ImportDialogComponent;

  protected readonly portfolioService = inject(PortfolioService);
  protected readonly transactionService = inject(TransactionService);
  protected readonly tickerState = inject(TickerStateService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly messages = inject(MessageService);
  private readonly confirmation = inject(ConfirmationService);

  protected readonly txDialogVisible = signal(false);
  protected readonly typeFilterValue = signal<TransactionType | null>(null);
  protected readonly typeFilterOptions = TYPE_FILTER_OPTIONS;
  protected readonly typeLabels = TYPE_LABELS;
  protected readonly typeSeverity = TYPE_SEVERITY;

  /** Sum of all position current values. Null until at least one price is available. */
  protected readonly portfolioTotal = computed<string | null>(() => {
    const holdings = this.transactionService.holdings();
    if (holdings.every((h) => h.currentValue === null)) return null;
    const total = holdings.reduce(
      (sum, h) => sum + (h.currentValue !== null ? parseFloat(h.currentValue) : 0),
      0,
    );
    return total.toFixed(2);
  });

  /** Sum of all position gain/loss amounts. Null until at least one price is available. */
  protected readonly portfolioGainLoss = computed<string | null>(() => {
    const holdings = this.transactionService.holdings();
    if (holdings.every((h) => h.gainLoss === null)) return null;
    const total = holdings.reduce(
      (sum, h) => sum + (h.gainLoss !== null ? parseFloat(h.gainLoss) : 0),
      0,
    );
    return total.toFixed(2);
  });

  /** Portfolio ID from route params. */
  private portfolioId = '';

  ngOnInit(): void {
    this.portfolioId = this.route.snapshot.paramMap.get('id') ?? '';

    this.portfolioService.loadById(this.portfolioId).subscribe({
      error: () =>
        this.messages.add({
          severity: 'error',
          summary: 'Not found',
          detail: 'Portfolio could not be loaded.',
        }),
    });

    this.transactionService.loadByPortfolio(this.portfolioId).subscribe({
      error: () =>
        this.messages.add({
          severity: 'error',
          summary: 'Load failed',
          detail: 'Could not load transactions.',
        }),
    });
  }

  ngOnDestroy(): void {
    this.transactionService.clear();
  }

  protected get filteredTransactions(): Transaction[] {
    const filter = this.typeFilterValue();
    const txs = this.transactionService.transactions();
    return filter ? txs.filter((t) => t.transaction_type === filter) : txs;
  }

  protected openTxDialog(): void {
    this.txDialogVisible.set(true);
  }

  protected onTxSubmitted(input: CreateTransactionInput): void {
    this.transactionService.create(this.portfolioId, input).subscribe({
      next: () => {
        this.txDialogVisible.set(false);
        this.messages.add({
          severity: 'success',
          summary: 'Transaction recorded',
          detail: `${TYPE_LABELS[input.transaction_type]} ${input.symbol}`,
        });
      },
      error: (err) => {
        this.messages.add({
          severity: 'error',
          summary: 'Failed to record transaction',
          detail: err?.error?.detail ?? 'An unexpected error occurred.',
        });
      },
    });
  }

  protected onTxCancelled(): void {
    this.txDialogVisible.set(false);
  }

  protected confirmDeleteTx(tx: Transaction): void {
    this.confirmation.confirm({
      message: `Delete this ${TYPE_LABELS[tx.transaction_type]} transaction for ${tx.symbol}?`,
      header: 'Confirm deletion',
      icon: 'pi pi-exclamation-triangle',
      acceptButtonStyleClass: 'p-button-danger',
      accept: () => this.deleteTx(tx),
    });
  }

  private deleteTx(tx: Transaction): void {
    this.transactionService.delete(this.portfolioId, tx.id).subscribe({
      next: () =>
        this.messages.add({
          severity: 'success',
          summary: 'Deleted',
          detail: 'Transaction removed.',
        }),
      error: () =>
        this.messages.add({
          severity: 'error',
          summary: 'Delete failed',
          detail: 'Could not delete the transaction.',
        }),
    });
  }

  /** Returns true when a string-encoded decimal is >= 0. Used for colour-coding. */
  protected isPositive(value: string | null): boolean {
    return value !== null && parseFloat(value) >= 0;
  }

  protected openImportDialog(): void {
    this.importDialog?.open();
  }

  protected onImported(result: any): void {
    // Reload transactions after successful import
    this.transactionService.loadByPortfolio(this.portfolioId).subscribe({
      error: () =>
        this.messages.add({
          severity: 'error',
          summary: 'Reload failed',
          detail: 'Could not reload transactions after import.',
        }),
    });
  }

  protected goBack(): void {
    this.router.navigate(['/portfolios']);
  }
}
