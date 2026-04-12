import {
  Component,
  input,
  computed,
  ChangeDetectionStrategy,
  viewChild,
  afterRenderEffect,
  effect,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ChartModule } from 'primeng/chart';
import { CardModule } from 'primeng/card';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import type { Holding } from '../../../portfolio/models/transaction.model';
import { Chart } from 'chart.js/auto';

/**
 * AllocationChartComponent displays portfolio allocation as a doughnut chart
 * using Chart.js via ng2-charts. Also displays allocation details in a table.
 *
 * Input: enrichedHoldings — holdings with live prices and market data.
 */
@Component({
  selector: 'app-allocation-chart',
  standalone: true,
  imports: [CommonModule, ChartModule, CardModule, TableModule, TagModule],
  templateUrl: './allocation-chart.component.html',
  styleUrl: './allocation-chart.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AllocationChartComponent {
  readonly holdings = input<Holding[]>([]);

  /** Allocation data computed from holdings. */
  readonly allocationData = computed(() => {
    const data = this.holdings()
      .map(h => ({
        symbol: h.symbol,
        value: h.currentValue ? parseFloat(h.currentValue) : 0,
      }))
      .filter(item => item.value > 0)
      .sort((a, b) => b.value - a.value);

    const total = data.reduce((sum, item) => sum + item.value, 0);
    return data.map(item => ({
      ...item,
      percentage: total > 0 ? ((item.value / total) * 100).toFixed(2) : '0.00',
    }));
  });

  /** Chart data for PrimeNG ChartModule. */
  readonly chartData = computed(() => {
    const data = this.allocationData();
    const colors = [
      '#3b82f6', // blue
      '#10b981', // green
      '#f59e0b', // amber
      '#ef4444', // red
      '#8b5cf6', // purple
      '#06b6d4', // cyan
    ];

    return {
      labels: data.map(item => item.symbol),
      datasets: [
        {
          data: data.map(item => parseFloat(item.value.toFixed(2))),
          backgroundColor: colors.slice(0, data.length),
          borderColor: '#fff',
          borderWidth: 2,
        },
      ],
    };
  });

  readonly chartOptions = {
    maintainAspectRatio: false,
    responsive: true,
    plugins: {
      legend: {
        position: 'bottom' as const,
      },
    },
  };
}
