import { Component, input, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CardModule } from 'primeng/card';
import { type Quote } from '../../../portfolio/models/market-data.model';

/**
 * KeyStatsCardComponent displays key market statistics for a ticker.
 * Shows day range, volume, open price, and previous close.
 *
 * Pure presentational component — no state or side effects.
 */
@Component({
  selector: 'app-key-stats-card',
  standalone: true,
  imports: [CommonModule, CardModule],
  templateUrl: './key-stats-card.component.html',
  styleUrls: ['./key-stats-card.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class KeyStatsCardComponent {
  readonly quote = input<Quote | null>(null);
  readonly livePrice = input<number | null>(null);
}
