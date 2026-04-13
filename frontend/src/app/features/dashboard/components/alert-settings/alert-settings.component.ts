import { Component, inject, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DialogModule } from 'primeng/dialog';
import { SelectModule } from 'primeng/select';
import { InputNumberModule } from 'primeng/inputnumber';
import { TableModule } from 'primeng/table';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { MessageService } from 'primeng/api';
import { PriceAlertService } from '../../../../core/alerts/price-alert.service';
import { UserPreferencesService } from '../../../../core/user-preferences.service';
import type { PortfolioAlertThreshold, AlertDirection } from '../../../../core/alerts/alert.model';

/**
 * AlertSettingsComponent allows users to configure alert preferences,
 * including browser notification permissions and portfolio/position alert thresholds.
 */
@Component({
  selector: 'app-alert-settings',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    ButtonModule,
    CardModule,
    DialogModule,
    SelectModule,
    InputNumberModule,
    TableModule,
    AutoCompleteModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <p-card>
      <ng-template pTemplate="header">
        <div class="text-lg font-semibold">Alert Settings</div>
      </ng-template>

      <!-- Browser Notifications Section -->
      <div class="mb-6 pb-6 border-b">
        <h3 class="text-base font-medium mb-3">Browser Notifications</h3>
        <p-button
          label="Enable Browser Notifications"
          icon="pi pi-bell"
          (onClick)="requestNotificationPermission()"
          [disabled]="notificationPermissionGranted()"
        />
        <p class="text-xs text-gray-500 mt-2">
          {{ notificationPermissionGranted() ? 'Browser notifications enabled' : 'Click to request permission' }}
        </p>
      </div>

      <!-- Portfolio Thresholds Section -->
      <div>
        <div class="flex justify-between items-center mb-3">
          <h3 class="text-base font-medium">Portfolio Alerts</h3>
          <p-button
            label="Add Threshold"
            icon="pi pi-plus"
            size="small"
            (onClick)="openAddDialog()"
          />
        </div>

        <!-- Thresholds Table -->
        <p-table
          [value]="preferences().thresholds"
          [tableStyle]="{ 'min-width': '50rem' }"
          [rows]="10"
          [paginator]="true"
          [globalFilterFields]="['type', 'symbol']"
        >
          <ng-template pTemplate="header">
            <tr>
              <th>Type</th>
              <th>Symbol</th>
              <th>Threshold</th>
              <th>Direction</th>
              <th>Actions</th>
            </tr>
          </ng-template>
          <ng-template pTemplate="body" let-threshold>
            <tr>
              <td>{{ formatThresholdType(threshold.type) }}</td>
              <td>{{ threshold.symbol || '—' }}</td>
              <td>{{ threshold.thresholdPercent }}%</td>
              <td>
                <span
                  [class]="
                    threshold.direction === 'above'
                      ? 'text-green-600 font-medium'
                      : 'text-red-600 font-medium'
                  "
                >
                  {{ threshold.direction === 'above' ? 'Up' : 'Down' }}
                </span>
              </td>
              <td>
                <p-button
                  icon="pi pi-trash"
                  severity="danger"
                  [text]="true"
                  size="small"
                  (onClick)="deleteThreshold(threshold.id)"
                />
              </td>
            </tr>
          </ng-template>
          <ng-template pTemplate="emptymessage">
            <tr>
              <td colspan="5" class="text-center py-4 text-gray-500">No thresholds configured</td>
            </tr>
          </ng-template>
        </p-table>
      </div>

      <!-- Add/Edit Threshold Dialog -->
      <p-dialog
        [visible]="isDialogOpen()"
        (visibleChange)="isDialogOpen.set($event)"
        [header]="'Add Alert Threshold'"
        [modal]="true"
        [style]="{ width: '50vw' }"
        [breakpoints]="{ '960px': '75vw', '640px': '90vw' }"
      >
        <form [formGroup]="thresholdForm" class="space-y-4">
          <!-- Type Select -->
          <div>
            <label for="type" class="block text-sm font-medium mb-2">Type</label>
            <p-select
              id="type"
              formControlName="type"
              [options]="thresholdTypes"
              optionLabel="label"
              optionValue="value"
              placeholder="Select type"
              class="w-full"
            />
          </div>

          <!-- Symbol (conditional) -->
          @if (thresholdForm.get('type')?.value === 'position_gain_loss') {
            <div>
              <label for="symbol" class="block text-sm font-medium mb-2">Symbol</label>
              <p-autoComplete
                id="symbol"
                formControlName="symbol"
                [suggestions]="filteredSymbols()"
                (completeMethod)="onSymbolSearch($event)"
                placeholder="AAPL, MSFT, etc."
                class="w-full"
                field="symbol"
              />
            </div>
          }

          <!-- Threshold Percentage -->
          <div>
            <label for="threshold" class="block text-sm font-medium mb-2">Threshold (%)</label>
            <p-inputNumber
              id="threshold"
              formControlName="thresholdPercent"
              [minFractionDigits]="1"
              [maxFractionDigits]="2"
              placeholder="-5.00"
              class="w-full"
            />
            <p class="text-xs text-gray-500 mt-1">
              Negative for loss (e.g., -5), positive for gain (e.g., 3)
            </p>
          </div>

          <!-- Direction Select -->
          <div>
            <label for="direction" class="block text-sm font-medium mb-2">Direction</label>
            <p-select
              id="direction"
              formControlName="direction"
              [options]="directions"
              optionLabel="label"
              optionValue="value"
              placeholder="Select direction"
              class="w-full"
            />
          </div>
        </form>

        <ng-template pTemplate="footer">
          <p-button label="Cancel" severity="secondary" (onClick)="closeDialog()" />
          <p-button
            label="Add"
            (onClick)="addThreshold()"
            [disabled]="!thresholdForm.valid"
          />
        </ng-template>
      </p-dialog>
    </p-card>
  `,
})
export class AlertSettingsComponent {
  private readonly priceAlertService = inject(PriceAlertService);
  private readonly preferencesService = inject(UserPreferencesService);
  private readonly messageService = inject(MessageService);
  private readonly formBuilder = inject(FormBuilder);

  readonly preferences = this.preferencesService.preferences;
  readonly isDialogOpen = signal(false);
  readonly filteredSymbols = signal<Array<{ symbol: string }>>([]);
  readonly notificationPermissionGranted = signal(false);

  readonly thresholdTypes = [
    { label: 'Portfolio Daily Change', value: 'portfolio_daily_change' },
    { label: 'Position Gain/Loss', value: 'position_gain_loss' },
  ];

  readonly directions = [
    { label: 'Up (Above)', value: 'above' as AlertDirection },
    { label: 'Down (Below)', value: 'below' as AlertDirection },
  ];

  // Popular symbols for autocomplete
  private readonly popularSymbols = [
    'AAPL', 'GOOGL', 'MSFT', 'AMZN', 'NVDA', 'META', 'TSLA',
    'BERKB', 'JPM', 'V', 'WMT', 'PG', 'XOM', 'COST', 'DIS',
  ];

  readonly thresholdForm: FormGroup = this.formBuilder.group({
    type: ['portfolio_daily_change', Validators.required],
    symbol: [''],
    thresholdPercent: [null, Validators.required],
    direction: ['below', Validators.required],
  });

  /**
   * Request browser notification permission from the user.
   * Must be called from a user gesture (button click).
   */
  async requestNotificationPermission(): Promise<void> {
    await this.priceAlertService.requestNotificationPermission();
    this.notificationPermissionGranted.set(true);
  }

  /**
   * Open the add threshold dialog.
   */
  openAddDialog(): void {
    this.thresholdForm.reset({ type: 'portfolio_daily_change', direction: 'below' });
    this.isDialogOpen.set(true);
  }

  /**
   * Close the dialog without saving.
   */
  closeDialog(): void {
    this.isDialogOpen.set(false);
  }

  /**
   * Add a new threshold to the user's preferences.
   */
  addThreshold(): void {
    if (!this.thresholdForm.valid) return;

    const formValue = this.thresholdForm.value;

    // Validate that position_gain_loss has a symbol
    if (formValue.type === 'position_gain_loss' && !formValue.symbol) {
      this.messageService.add({
        severity: 'error',
        summary: 'Symbol required',
        detail: 'Please select a symbol for position alerts.',
      });
      return;
    }

    const newThreshold: PortfolioAlertThreshold = {
      id: `threshold-${Date.now()}`,
      type: formValue.type,
      symbol: formValue.symbol || undefined,
      thresholdPercent: formValue.thresholdPercent,
      direction: formValue.direction,
      fired: false,
    };

    const updatedThresholds = [...this.preferences().thresholds, newThreshold];
    this.preferencesService.saveThresholds(updatedThresholds).subscribe({
      next: () => {
        this.closeDialog();
        this.messageService.add({
          severity: 'success',
          summary: 'Threshold added',
          detail: 'Your alert threshold has been saved.',
        });
      },
      error: () => {
        this.messageService.add({
          severity: 'error',
          summary: 'Save failed',
          detail: 'Could not save the threshold.',
        });
      },
    });
  }

  /**
   * Delete a threshold by ID.
   */
  deleteThreshold(id: string): void {
    const updatedThresholds = this.preferences().thresholds.filter((t) => t.id !== id);
    this.preferencesService.saveThresholds(updatedThresholds).subscribe({
      next: () => {
        this.messageService.add({
          severity: 'success',
          summary: 'Threshold deleted',
          detail: 'Your alert threshold has been removed.',
        });
      },
      error: () => {
        this.messageService.add({
          severity: 'error',
          summary: 'Delete failed',
          detail: 'Could not delete the threshold.',
        });
      },
    });
  }

  /**
   * Handle symbol search in the autocomplete.
   */
  onSymbolSearch(event: { query: string }): void {
    const query = event.query.toUpperCase();
    const filtered = this.popularSymbols
      .filter((s) => s.includes(query))
      .map((symbol) => ({ symbol }));
    this.filteredSymbols.set(filtered);
  }

  /**
   * Format threshold type for display.
   */
  formatThresholdType(type: string): string {
    return type === 'portfolio_daily_change' ? 'Portfolio Daily Change' : 'Position Gain/Loss';
  }
}
