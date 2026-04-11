import { inject, Injectable, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap } from 'rxjs/operators';
import { Observable } from 'rxjs';
import { environment } from '../../../../environments/environment';
import type {
  Transaction,
  CreateTransactionInput,
  Holding,
} from '../models/transaction.model';

const portfolioBase = (portfolioId: string) =>
  `${environment.apiBaseUrl}/portfolios/${portfolioId}/transactions`;

/**
 * Derives current holdings from a list of transactions.
 *
 * Holdings are never stored — they are always computed from the transaction
 * ledger. Only symbols with a positive net quantity appear in the result.
 *
 * Cost basis accounts only for buy and reinvested_dividend transactions.
 * Dividends are income events and do not affect cost basis.
 *
 * Exported as a pure function so it can be tested in isolation.
 */
export function deriveHoldings(transactions: Transaction[]): Holding[] {
  const bySymbol = new Map<
    string,
    { netQty: number; totalCost: number }
  >();

  for (const tx of transactions) {
    const sym = tx.symbol;
    if (!bySymbol.has(sym)) {
      bySymbol.set(sym, { netQty: 0, totalCost: 0 });
    }
    const entry = bySymbol.get(sym)!;

    switch (tx.transaction_type) {
      case 'buy':
      case 'reinvested_dividend':
        entry.netQty += Number(tx.quantity ?? 0);
        entry.totalCost += Number(tx.total_amount);
        break;
      case 'sell':
        entry.netQty -= Number(tx.quantity ?? 0);
        break;
      case 'dividend':
        // Cash dividend — no impact on share quantity or cost basis.
        break;
    }
  }

  const holdings: Holding[] = [];
  for (const [symbol, { netQty, totalCost }] of bySymbol) {
    if (netQty <= 0) continue;
    const avgCostBasis = netQty > 0 ? totalCost / netQty : 0;
    holdings.push({
      symbol,
      quantity: netQty.toFixed(6).replace(/\.?0+$/, ''),
      avgCostBasis: avgCostBasis.toFixed(2),
      totalCost: totalCost.toFixed(2),
      // Market data fields are null until live prices arrive from TickerStateService.
      currentPrice: null,
      currentValue: null,
      gainLoss: null,
      gainLossPercent: null,
    });
  }

  return holdings.sort((a, b) => a.symbol.localeCompare(b.symbol));
}

/**
 * TransactionService manages transaction CRUD and derives live holdings.
 *
 * The `holdings` computed signal re-derives whenever the transaction list
 * changes — no separate API call or stored state required.
 */
@Injectable({ providedIn: 'root' })
export class TransactionService {
  private readonly http = inject(HttpClient);

  private readonly _transactions = signal<Transaction[]>([]);
  private readonly _loading = signal(false);

  /** Current transaction list for the active portfolio. */
  readonly transactions = this._transactions.asReadonly();

  /** True while any HTTP request is in flight. */
  readonly loading = this._loading.asReadonly();

  /**
   * Holdings derived from the current transaction list.
   *
   * Re-computes automatically whenever transactions change.
   * Never stored in the database — ADR 007.
   */
  readonly holdings = computed<Holding[]>(() =>
    deriveHoldings(this._transactions()),
  );

  /** Loads all transactions for a portfolio and resets the signal. */
  loadByPortfolio(portfolioId: string): Observable<Transaction[]> {
    this._loading.set(true);
    return this.http.get<Transaction[]>(portfolioBase(portfolioId)).pipe(
      tap({
        next: (data) => {
          this._transactions.set(data);
          this._loading.set(false);
        },
        error: () => this._loading.set(false),
      }),
    );
  }

  /** Records a new transaction and prepends it to the signal list. */
  create(
    portfolioId: string,
    input: CreateTransactionInput,
  ): Observable<Transaction> {
    return this.http
      .post<Transaction>(portfolioBase(portfolioId), input)
      .pipe(tap((tx) => this._transactions.update((txs) => [tx, ...txs])));
  }

  /** Deletes a transaction and removes it from the signal list. */
  delete(portfolioId: string, txId: string): Observable<void> {
    return this.http
      .delete<void>(`${portfolioBase(portfolioId)}/${txId}`)
      .pipe(
        tap(() =>
          this._transactions.update((txs) => txs.filter((t) => t.id !== txId)),
        ),
      );
  }

  /** Clears the transaction list (call when navigating away from a portfolio). */
  clear(): void {
    this._transactions.set([]);
  }
}
