import { Component, input, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';
import type { Holding } from '../../../portfolio/models/transaction.model';

/**
 * PositionSummaryCardComponent displays the user's position in a ticker.
 * Shows quantity, cost basis, current value, gain/loss, and holding period.
 *
 * Pure presentational component — no state or side effects.
 */
@Component({
  selector: 'app-position-summary-card',
  standalone: true,
  imports: [CommonModule, CardModule, TagModule],
  templateUrl: './position-summary-card.component.html',
  styleUrls: ['./position-summary-card.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PositionSummaryCardComponent {
  readonly holding = input<Holding | null>(null);
  readonly holdingPeriod = input<string | null>(null);

  // Compute number of days held
  readonly daysHeld = computed(() => {
    const startDate = this.holdingPeriod();
    if (!startDate) return null;

    const start = new Date(startDate);
    const today = new Date();
    const diffMs = today.getTime() - start.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    return diffDays;
  });

  // Compute formatted holding period text
  readonly holdingPeriodText = computed(() => {
    const startDate = this.holdingPeriod();
    const days = this.daysHeld();
    if (!startDate) return null;

    const dateObj = new Date(startDate);
    const year = dateObj.getFullYear();
    const month = String(dateObj.getMonth() + 1).padStart(2, '0');
    const day = String(dateObj.getDate()).padStart(2, '0');
    const formattedDate = `${year}-${month}-${day}`;

    return `Since ${formattedDate} (${days} days)`;
  });

  // Compute gain/loss class for styling
  readonly gainLossClass = computed(() => {
    const h = this.holding();
    if (!h || h.gainLoss === null) return '';
    const gainNum = parseFloat(h.gainLoss);
    return gainNum >= 0 ? 'text-success-600' : 'text-danger-600';
  });

  // Compute gain/loss percent class for styling
  readonly gainLossPercentClass = computed(() => {
    const h = this.holding();
    if (!h || h.gainLossPercent === null) return '';
    return h.gainLossPercent >= 0 ? 'text-success-600' : 'text-danger-600';
  });

  // Expose parseFloat for template use
  parseFloat = parseFloat;
}
