import {
  Component,
  computed,
  effect,
  inject,
  signal,
  DestroyRef,
  ChangeDetectionStrategy,
  OnInit,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { CardModule } from 'primeng/card';
import { ButtonModule } from 'primeng/button';
import { TagModule } from 'primeng/tag';
import { TooltipModule } from 'primeng/tooltip';
import { MarketDataService } from '../../../../core/market-data.service';
import { TickerStateService } from '../../../../core/ticker-state.service';
import { ThemeService } from '../../../../core/theme.service';
import { TransactionService, deriveHoldings, enrichHoldingsWithPrices } from '../../../portfolio/services/transaction.service';
import { PortfolioService } from '../../../portfolio/services/portfolio.service';
import type { Quote, Bar, Timeframe } from '../../../portfolio/models/market-data.model';
import type { Transaction, Holding } from '../../../portfolio/models/transaction.model';
import { computeHoldingPeriod } from './ticker-detail.utils';
import { TickerChartComponent } from '../../components/ticker-chart/ticker-chart.component';
import { KeyStatsCardComponent } from '../../components/key-stats-card/key-stats-card.component';
import { PositionSummaryCardComponent } from '../../components/position-summary-card/position-summary-card.component';
import { TickerTransactionsTableComponent } from '../../components/ticker-transactions-table/ticker-transactions-table.component';
import { ConnectionStatusComponent } from '../../../../shared/components/connection-status/connection-status.component';

@Component({
  selector: 'app-ticker-detail',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    CardModule,
    ButtonModule,
    TagModule,
    TooltipModule,
    TickerChartComponent,
    KeyStatsCardComponent,
    PositionSummaryCardComponent,
    TickerTransactionsTableComponent,
    ConnectionStatusComponent,
  ],
  templateUrl: './ticker-detail.component.html',
  styleUrls: ['./ticker-detail.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TickerDetailComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly marketDataService = inject(MarketDataService);
  readonly tickerStateService = inject(TickerStateService);
  private readonly themeService = inject(ThemeService);
  private readonly portfolioService = inject(PortfolioService);
  private readonly transactionService = inject(TransactionService);
  private readonly destroyRef = inject(DestroyRef);

  constructor() {
    // Effect that runs whenever selectedTimeframe changes.
    // Fetches fresh bar data for the new timeframe.
    effect(() => {
      const tf = this.selectedTimeframe();
      const sym = this.symbol();
      if (!sym) return;

      this.loadBars(sym, tf);
    });
  }

  // Route param
  readonly symbol = signal<string>('');

  // Remote data state
  readonly quote = signal<Quote | null>(null);
  readonly bars = signal<Bar[]>([]);
  readonly barsLoading = signal(false);
  readonly quoteLoading = signal(true);

  // Time range selector
  readonly selectedTimeframe = signal<Timeframe>('1M');
  readonly timeframes: Timeframe[] = ['1D', '1W', '1M', '3M', '1Y', 'ALL'];

  // Transactions (local accumulation across all portfolios)
  readonly allTransactions = signal<Transaction[]>([]);

  // Ticker real-time state from WebSocket
  readonly tickerState = computed(() =>
    this.tickerStateService.tickers()[this.symbol()] ?? null
  );
  readonly livePrice = computed(() => this.tickerState()?.currentPrice ?? null);

  // Filter transactions to this symbol
  readonly symbolTransactions = computed(() =>
    this.allTransactions().filter((tx) => tx.symbol === this.symbol())
  );

  // Derive holding for this symbol using existing pure functions
  readonly symbolHolding = computed<Holding | null>(() => {
    const txs = this.symbolTransactions();
    const holdings = deriveHoldings(txs);
    if (!holdings.length) return null;

    const price = this.livePrice();
    const prices = price !== null ? { [this.symbol()]: price } : {};
    return enrichHoldingsWithPrices(holdings, prices)[0] ?? null;
  });

  // Holding period (earliest transaction date)
  readonly holdingPeriod = computed(() =>
    computeHoldingPeriod(this.symbolTransactions())
  );

  // Computed for UI state
  readonly isDark = computed(() => this.themeService.isDark());

  ngOnInit(): void {
    // Extract symbol from route params
    const sym = this.route.snapshot.paramMap.get('symbol');
    if (sym) {
      this.symbol.set(sym);

      // Subscribe to WebSocket for this symbol
      this.tickerStateService.subscribe([sym]);

      // Load quote snapshot
      this.loadQuote(sym);

      // Load initial bars for 1M timeframe
      this.loadBars(sym, '1M');

      // Load all transactions across all portfolios
      this.loadAllTransactions();
    }

    // Cleanup on destroy
    this.destroyRef.onDestroy(() => {
      const sym = this.symbol();
      if (sym) {
        this.tickerStateService.unsubscribe([sym]);
      }
    });
  }

  /**
   * Load quote snapshot for the symbol.
   */
  private loadQuote(symbol: string): void {
    this.quoteLoading.set(true);
    this.marketDataService.getQuote(symbol).subscribe({
      next: (quote) => {
        this.quote.set(quote);
        this.quoteLoading.set(false);
      },
      error: () => {
        this.quoteLoading.set(false);
      },
    });
  }

  /**
   * Load historical bars for the symbol and timeframe.
   */
  private loadBars(symbol: string, timeframe: Timeframe): void {
    this.barsLoading.set(true);
    this.marketDataService.getHistoricalBars(symbol, timeframe).subscribe({
      next: (bars) => {
        this.bars.set(bars);
        this.barsLoading.set(false);
      },
      error: () => {
        this.barsLoading.set(false);
      },
    });
  }

  /**
   * Load all transactions across all portfolios.
   * If TransactionService already has transactions loaded, use those.
   * Otherwise, load all portfolios and their transactions.
   */
  private loadAllTransactions(): void {
    const existing = this.transactionService.transactions();
    if (existing.length > 0) {
      // Already loaded from a previous portfolio view; seed local signal
      this.allTransactions.set(existing);
    } else {
      // Load all portfolios and their transactions
      this.portfolioService.loadAll().subscribe({
        next: (portfolios) => {
          const txs: Transaction[] = [];
          let completed = 0;

          if (portfolios.length === 0) {
            this.allTransactions.set([]);
            return;
          }

          portfolios.forEach((portfolio) => {
            this.transactionService.loadByPortfolio(portfolio.id).subscribe({
              next: (txsList) => {
                txs.push(...txsList);
                completed++;
                if (completed === portfolios.length) {
                  this.allTransactions.set(txs);
                }
              },
            });
          });
        },
      });
    }
  }

  /**
   * User clicked a time range button.
   */
  selectTimeframe(tf: Timeframe): void {
    this.selectedTimeframe.set(tf);
  }

  /**
   * Navigate back to the previous page.
   */
  goBack(): void {
    window.history.back();
  }
}
