import {
  Component,
  input,
  inject,
  viewChild,
  DestroyRef,
  ElementRef,
  effect,
  ChangeDetectionStrategy,
  afterRenderEffect,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import type { Bar } from '../../../portfolio/models/market-data.model';
import {
  createChart,
  type IChartApi,
  type ISeriesApi,
  CandlestickSeries,
  HistogramSeries,
  type CandlestickData,
  type HistogramData,
  type Time,
} from 'lightweight-charts';

/**
 * TickerChartComponent renders a professional candlestick chart with volume histogram
 * using Lightweight Charts v5. The chart updates in real-time as prices stream in,
 * and responds to dark mode theme changes.
 *
 * Inputs are signals to ensure OnPush change detection compatibility and enable
 * reactive effects for data updates and theme changes.
 */
@Component({
  selector: 'app-ticker-chart',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './ticker-chart.component.html',
  styleUrls: ['./ticker-chart.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TickerChartComponent {
  // Inputs
  readonly bars = input.required<Bar[]>();
  readonly livePrice = input<number | null>(null);
  readonly isDark = input.required<boolean>();

  // DOM element reference
  readonly chartEl = viewChild.required<ElementRef>('chartContainer');

  private readonly destroyRef = inject(DestroyRef);

  // Chart instances
  private chartInstance: IChartApi | null = null;
  private candleSeries: ISeriesApi<'Candlestick'> | null = null;
  private volumeSeries: ISeriesApi<'Histogram'> | null = null;

  constructor() {
    afterRenderEffect(() => {
      this.initializeChart();
    });
  }

  /**
   * Initialize the Lightweight Charts instance with candlestick + volume histogram.
   * Called via afterRenderEffect to ensure the DOM element is painted and ready.
   */
  private initializeChart(): void {
    const el = this.chartEl().nativeElement as HTMLElement;

    // Create chart instance
    this.chartInstance = createChart(el, {
      width: el.clientWidth,
      height: 480,
      layout: {
        background: { color: 'transparent' },
        textColor: this.isDark() ? '#e2e8f0' : '#1e293b',
        fontFamily: 'system-ui, -apple-system, sans-serif',
      },
      timeScale: {
        timeVisible: true,
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

    // Candlestick series on default pane
    this.candleSeries = this.chartInstance.addSeries(CandlestickSeries, {
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderVisible: false,
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444',
    });

    // Volume histogram on separate price scale
    this.volumeSeries = this.chartInstance.addSeries(HistogramSeries, {
      color: '#94a3b8',
      priceFormat: { type: 'volume' },
      priceScaleId: 'volume',
    });

    // Set volume series to a separate price scale (bottom/secondary)
    this.chartInstance.priceScale('volume').applyOptions({
      scaleMargins: { top: 0.7, bottom: 0 },
    });

    // Set initial data from bars signal
    this.updateSeriesData(this.bars());

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
      this.chartInstance?.remove();
      this.chartInstance = null;
    });

    // Reactive effect: update series when bars signal changes
    effect(() => {
      const newBars = this.bars();
      if (newBars.length > 0) {
        this.updateSeriesData(newBars);
      }
    });

    // Reactive effect: update last candle when live price arrives
    effect(() => {
      const price = this.livePrice();
      const barList = this.bars();
      if (price === null || !barList.length || !this.candleSeries) return;

      const lastBar = barList[barList.length - 1];
      const updatedCandle: CandlestickData = {
        time: this.barTimestamp(lastBar),
        open: lastBar.open,
        high: Math.max(lastBar.high, price),
        low: Math.min(lastBar.low, price),
        close: price,
      };

      this.candleSeries.update(updatedCandle);
    });

    // Reactive effect: update chart theme when dark mode changes
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
  }

  /**
   * Convert a Bar to Lightweight Charts timestamp (Unix seconds).
   */
  private barTimestamp(bar: Bar): Time {
    return (new Date(bar.timestamp).getTime() / 1000) as Time;
  }

  /**
   * Convert a Bar to CandlestickData for the candlestick series.
   */
  private barToCandle(bar: Bar): CandlestickData {
    return {
      time: this.barTimestamp(bar),
      open: bar.open,
      high: bar.high,
      low: bar.low,
      close: bar.close,
    };
  }

  /**
   * Convert a Bar to HistogramData for the volume series.
   * Color is green if close >= open, red otherwise (semi-transparent).
   */
  private barToVolume(bar: Bar): HistogramData {
    const isUp = bar.close >= bar.open;
    return {
      time: this.barTimestamp(bar),
      value: bar.volume,
      color: isUp ? '#22c55e40' : '#ef444440',
    };
  }

  /**
   * Update both series with new bar data.
   * Calls setData() to replace all candles (used when timeframe changes).
   */
  private updateSeriesData(bars: Bar[]): void {
    const candles = bars.map((b) => this.barToCandle(b));
    const volumes = bars.map((b) => this.barToVolume(b));

    if (this.candleSeries) {
      this.candleSeries.setData(candles);
    }
    if (this.volumeSeries) {
      this.volumeSeries.setData(volumes);
    }

    // Auto-scale to fit all data
    if (this.chartInstance) {
      this.chartInstance.timeScale().fitContent();
    }
  }
}
