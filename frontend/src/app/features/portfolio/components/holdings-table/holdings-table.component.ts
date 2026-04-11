import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TableModule } from 'primeng/table';
import { Tag } from 'primeng/tag';
import type { Holding } from '../../models/transaction.model';

/**
 * HoldingsTableComponent displays a read-only table of derived holdings.
 *
 * Holdings are computed from transactions — never stored (ADR 007).
 * Live prices and gain/loss will be added in Phase 5 once market data
 * integration is in place.
 */
@Component({
  selector: 'app-holdings-table',
  standalone: true,
  imports: [TableModule, Tag],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './holdings-table.component.html',
})
export class HoldingsTableComponent {
  /** Holdings derived from the current transaction list. */
  readonly holdings = input.required<Holding[]>();
}
