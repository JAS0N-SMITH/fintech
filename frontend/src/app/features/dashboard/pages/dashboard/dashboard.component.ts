import {
  Component,
  ChangeDetectionStrategy,
  OnInit,
  OnDestroy,
  inject,
  signal,
  computed,
  effect,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { CardModule } from 'primeng/card';
import { TabsModule } from 'primeng/tabs';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { ChartModule } from 'primeng/chart';
import { AllocationChartComponent } from '../../components/allocation-chart/allocation-chart.component';
import { PortfolioService } from '../../../portfolio/services/portfolio.service';
import { TransactionService, deriveHoldings, enrichHoldingsWithPrices } from '../../../portfolio/services/transaction.service';
import { TickerStateService } from '../../../../core/ticker-state.service';
import type { Portfolio } from '../../../portfolio/models/portfolio.model';

/**
 * DashboardComponent displays a portfolio overview with summary cards,
 * allocation chart, performance chart, and top movers.
 *
 * Data is loaded from all user portfolios and aggregated client-side.
 * Market data comes from live WebSocket ticks via TickerStateService.
 */
@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    TabsModule,
    TableModule,
    TagModule,
    ChartModule,
    AllocationChartComponent,
  ],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DashboardComponent implements OnInit, OnDestroy {
  private readonly portfolioService: PortfolioService = inject(PortfolioService);
  private readonly transactionService: TransactionService = inject(TransactionService);
  private readonly tickerStateService: TickerStateService = inject(TickerStateService);

  readonly portfolios = this.portfolioService.portfolios;
  readonly isLoading = signal(false);

  /** All transactions across all portfolios. */
  private readonly allTransactions = signal<any[]>([]);

  /** All holdings derived from all transactions, aggregated by symbol. */
  readonly allHoldings = computed(() => {
    const txs = this.allTransactions();
    if (txs.length === 0) return [];
    return deriveHoldings(txs);
  });

  /** All holdings enriched with live market prices. */
  readonly enrichedHoldings = computed(() => {
    const holdings = this.allHoldings();
    const tickers = this.tickerStateService.tickers();
    const prices: Record<string, number> = {};
    for (const sym of Object.keys(tickers)) {
      const price = tickers[sym].currentPrice;
      if (price !== null) prices[sym] = price;
    }
    return enrichHoldingsWithPrices(holdings, prices);
  });

  /** Total portfolio value across all holdings. */
  readonly totalPortfolioValue = computed(() => {
    return this.enrichedHoldings()
      .reduce((sum, h) => sum + (h.currentValue ? parseFloat(h.currentValue) : 0), 0)
      .toFixed(2);
  });

  /** Total unrealized gain/loss across all holdings. */
  readonly totalUnrealizedGainLoss = computed(() => {
    return this.enrichedHoldings()
      .reduce((sum, h) => sum + (h.gainLoss ? parseFloat(h.gainLoss) : 0), 0)
      .toFixed(2);
  });

  /** Total unrealized gain/loss percentage. */
  readonly totalUnrealizedGainLossPercent = computed(() => {
    const gainLoss = parseFloat(this.totalUnrealizedGainLoss());
    const totalCost = this.enrichedHoldings()
      .reduce((sum, h) => sum + (h.totalCost ? parseFloat(h.totalCost) : 0), 0);
    return totalCost !== 0 ? ((gainLoss / totalCost) * 100).toFixed(2) : '0.00';
  });

  /** Day gain/loss: sum of (quantity × (currentPrice - previousClose)) for all holdings. */
  readonly dayGainLoss = computed(() => {
    return this.enrichedHoldings()
      .reduce((sum, h) => {
        if (!h.currentPrice) return sum;
        const ticker = this.tickerStateService.tickers()[h.symbol];
        if (!ticker || ticker.previousClose === null || ticker.previousClose === undefined) return sum;
        const dayChange = parseFloat(h.quantity) * (h.currentPrice - ticker.previousClose);
        return sum + dayChange;
      }, 0)
      .toFixed(2);
  });

  /** Day gain/loss percentage. */
  readonly dayGainLossPercent = computed(() => {
    const dayGainLoss = parseFloat(this.dayGainLoss());
    const totalCost = this.enrichedHoldings()
      .reduce((sum, h) => sum + (h.totalCost ? parseFloat(h.totalCost) : 0), 0);
    return totalCost !== 0 ? ((dayGainLoss / totalCost) * 100).toFixed(2) : '0.00';
  });

  /** Top 5 gainers (sorted by day change % descending). */
  readonly topGainers = computed(() => {
    return this.enrichedHoldings()
      .map(h => {
        const ticker = this.tickerStateService.tickers()[h.symbol];
        const dayChange = h.currentPrice && ticker && ticker.previousClose !== null
          ? parseFloat(h.quantity) * (h.currentPrice - ticker.previousClose)
          : 0;
        const dayChangePercent = ticker && ticker.previousClose !== null && ticker.previousClose !== 0
          ? ((h.currentPrice! - ticker.previousClose) / ticker.previousClose) * 100
          : 0;
        return { ...h, dayChange, dayChangePercent };
      })
      .sort((a: any, b: any) => b.dayChangePercent - a.dayChangePercent)
      .slice(0, 5);
  });

  /** Top 5 losers (sorted by day change % ascending). */
  readonly topLosers = computed(() => {
    return this.enrichedHoldings()
      .map(h => {
        const ticker = this.tickerStateService.tickers()[h.symbol];
        const dayChange = h.currentPrice && ticker && ticker.previousClose !== null
          ? parseFloat(h.quantity) * (h.currentPrice - ticker.previousClose)
          : 0;
        const dayChangePercent = ticker && ticker.previousClose !== null && ticker.previousClose !== 0
          ? ((h.currentPrice! - ticker.previousClose) / ticker.previousClose) * 100
          : 0;
        return { ...h, dayChange, dayChangePercent };
      })
      .sort((a: any, b: any) => a.dayChangePercent - b.dayChangePercent)
      .slice(0, 5);
  });

  ngOnInit(): void {
    this.loadAllTransactions();
  }

  ngOnDestroy(): void {
    this.transactionService.clear();
  }

  /** Exposing parseFloat to the template. */
  parseFloat = parseFloat;

  private loadAllTransactions(): void {
    this.isLoading.set(true);
    this.portfolioService.loadAll().subscribe({
      next: (portfolios: Portfolio[]) => {
        // Load transactions for all portfolios
        const requests = portfolios.map(p =>
          this.transactionService.loadByPortfolio(p.id).toPromise()
        );

        Promise.all(requests).then((results) => {
          const allTxs: any[] = [];
          for (const result of results) {
            if (result) {
              allTxs.push(...result);
            }
          }
          this.allTransactions.set(allTxs);

          // Subscribe to live prices for all symbols
          const symbols = [...new Set(allTxs.map(tx => tx.symbol))];
          if (symbols.length > 0) {
            this.tickerStateService.subscribe(symbols);
          }

          this.isLoading.set(false);
        }).catch(() => {
          this.isLoading.set(false);
        });
      },
      error: () => {
        this.isLoading.set(false);
      },
    });
  }
}
