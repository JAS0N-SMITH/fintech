import {
  Component,
  input,
  inject,
  viewChild,
  DestroyRef,
  ElementRef,
  signal,
  computed,
  effect,
  ChangeDetectionStrategy,
  afterRenderEffect,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { forkJoin, of } from 'rxjs';
import { map, catchError } from 'rxjs/operators';
import type { Subscription } from 'rxjs';
import {
  createChart,
  type IChartApi,
  type ISeriesApi,
  LineSeries,
  type LineData,
  type Time,
} from 'lightweight-charts';
import { MarketDataService } from '../../../../core/market-data.service';
import { ThemeService } from '../../../../core/theme.service';
import type { Transaction } from '../../../portfolio/models/transaction.model';
import {
  derivePortfolioValues,
  type LineDataPoint,
  type SymbolBars,
} from './portfolio-performance-chart.utils';

type PortfolioTimeframe = '1M' | '3M' | '1Y';

/**
 * PortfolioPerformanceChartComponent displays total portfolio value over time
 * using a line chart from Lightweight Charts v5.
 *
 * The component accepts a list of transactions, fetches historical OHLCV bars
 * for all held symbols, derives daily portfolio values, and plots them as a line.
 *
 * The user can select from 1M, 3M, or 1Y time ranges via tab buttons.
 */
@Component({
  selector: 'app-portfolio-performance-chart',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './portfolio-performance-chart.component.html',
  styleUrl: './portfolio-performance-chart.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PortfolioPerformanceChartComponent {
  // Inputs
  readonly transactions = input.required<Transaction[]>();

  // DOM reference (optional, only exists when chart is not loading/empty/error)
  readonly chartEl = viewChild<ElementRef>('chartContainer');

  // Services
  private readonly marketDataService = inject(MarketDataService);
  private readonly themeService = inject(ThemeService);
  private readonly destroyRef = inject(DestroyRef);

  // State
  readonly selectedTimeframe = signal<PortfolioTimeframe>('1M');
  readonly isLoading = signal(false);
  readonly hasError = signal(false);
  readonly chartPoints = signal<LineDataPoint[]>([]);

  // Computed
  readonly isEmpty = computed(() => this.transactions().length === 0);
  readonly isDark = computed(() => this.themeService.isDark());

  // Chart instances
  private chartInstance: IChartApi | null = null;
  private lineSeries: ISeriesApi<'Line'> | null = null;

  // Active RxJS subscription (for cleanup)
  private activeSubscription: Subscription | null = null;

  // Timeframe options for the UI
  readonly timeframeOptions: PortfolioTimeframe[] = ['1M', '3M', '1Y'];

  constructor() {
    // Effect 1: Trigger fetch when transactions or timeframe changes
    effect(() => {
      const txs = this.transactions();
      const tf = this.selectedTimeframe();
      this.fetchAndDerive(txs, tf);
    });

    // Effect 2: Update line series when chart points change (after chart is initialized)
    effect(() => {
      const points = this.chartPoints();
      if (!this.lineSeries) return;

      this.lineSeries.setData(points);
      if (points.length > 0 && this.chartInstance) {
        this.chartInstance.timeScale().fitContent();
      }
    });

    // Effect 3: Update chart theme when dark mode changes (after chart is initialized)
    effect(() => {
      const dark = this.isDark();
      if (!this.chartInstance) return;

      this.chartInstance.applyOptions({
        layout: {
          background: { color: 'transparent' },
          textColor: dark ? '#e2e8f0' : '#1e293b',
        },
        grid: {
          vertLines: { color: dark ? '#334155' : '#e2e8f0' },
          horzLines: { color: dark ? '#334155' : '#e2e8f0' },
        },
      });

      if (this.chartInstance.priceScale('right')) {
        this.chartInstance.priceScale('right').applyOptions({
          textColor: dark ? '#94a3b8' : '#64748b',
        });
      }
    });

    // Initialize chart once DOM is ready
    afterRenderEffect(() => {
      // Guard: only initialize if we have data and container exists
      if (!this.isEmpty() && !this.isLoading() && !this.hasError()) {
        const container = this.chartEl();
        if (container && container.nativeElement) {
          this.initializeChart();
        }
      }
    });
  }

  /**
   * User selected a different timeframe tab.
   */
  selectTimeframe(tf: PortfolioTimeframe): void {
    this.selectedTimeframe.set(tf);
  }

  /**
   * Fetch historical bars for all unique symbols in transactions,
   * derive portfolio values, and update the chart.
   */
  private fetchAndDerive(txs: Transaction[], tf: PortfolioTimeframe): void {
    // Cancel any in-flight subscription
    this.activeSubscription?.unsubscribe();

    // Guard: no transactions
    if (txs.length === 0) {
      this.chartPoints.set([]);
      return;
    }

    this.isLoading.set(true);
    this.hasError.set(false);

    // Determine date range
    const { start, end } = this.dateRangeForTimeframe(tf);

    // Collect unique symbols from transactions
    const symbols = [...new Set(txs.map((tx) => tx.symbol))];

    if (symbols.length === 0) {
      this.chartPoints.set([]);
      this.isLoading.set(false);
      return;
    }

    // Fetch bars in parallel for all symbols
    this.activeSubscription = forkJoin(
      symbols.map((sym) =>
        this.marketDataService
          .getHistoricalBars(sym, '1D', start, end)
          .pipe(
            map((bars) => ({ sym, bars })),
            catchError(() => of({ sym, bars: [] })) // Tolerate individual failures
          )
      )
    ).subscribe({
      next: (results: SymbolBars[]) => {
        // Derive portfolio values from transactions + bars
        const points = derivePortfolioValues(txs, results);
        this.chartPoints.set(points);
        this.isLoading.set(false);
      },
      error: () => {
        this.hasError.set(true);
        this.isLoading.set(false);
      },
    });
  }

  /**
   * Compute date range (start, end) for the selected timeframe.
   * Returns RFC3339 format required by the backend API.
   */
  private dateRangeForTimeframe(
    tf: PortfolioTimeframe
  ): { start: string; end: string } {
    const today = new Date();
    const end = today.toISOString(); // RFC3339 format

    let daysBack = 30; // Default 1M
    if (tf === '3M') {
      daysBack = 90;
    } else if (tf === '1Y') {
      daysBack = 365;
    }

    const start = new Date(today.getTime() - daysBack * 24 * 60 * 60 * 1000)
      .toISOString(); // RFC3339 format

    return { start, end };
  }

  /**
   * Initialize the Lightweight Charts instance with a line series.
   * Called via afterRenderEffect to ensure DOM is ready.
   */
  private initializeChart(): void {
    const container = this.chartEl();
    if (!container) return;

    const el = container.nativeElement as HTMLElement;

    this.chartInstance = createChart(el, {
      width: el.clientWidth,
      height: 300,
      layout: {
        background: { color: 'transparent' },
        textColor: this.isDark() ? '#e2e8f0' : '#1e293b',
        fontFamily: 'system-ui, -apple-system, sans-serif',
      },
      timeScale: {
        timeVisible: false,
        secondsVisible: false,
      },
      rightPriceScale: {
        visible: true,
        textColor: this.isDark() ? '#94a3b8' : '#64748b',
      },
      grid: {
        vertLines: { color: this.isDark() ? '#334155' : '#e2e8f0' },
        horzLines: { color: this.isDark() ? '#334155' : '#e2e8f0' },
      },
    });

    // Add line series for portfolio value
    this.lineSeries = this.chartInstance.addSeries(LineSeries, {
      color: '#3b82f6', // Tailwind blue-500
      lineWidth: 2,
      priceFormat: { type: 'price', precision: 2, minMove: 0.01 },
    });

    // Set initial data from signal
    const points = this.chartPoints();
    if (points.length > 0) {
      this.lineSeries.setData(points);
      this.chartInstance.timeScale().fitContent();
    }

    // Responsive resize observer
    const ro = new ResizeObserver(() => {
      if (this.chartInstance) {
        this.chartInstance.applyOptions({ width: el.clientWidth });
      }
    });
    ro.observe(el);

    // Cleanup on destroy
    this.destroyRef.onDestroy(() => {
      ro.disconnect();
      this.activeSubscription?.unsubscribe();
      this.chartInstance?.remove();
      this.chartInstance = null;
    });
  }
}
