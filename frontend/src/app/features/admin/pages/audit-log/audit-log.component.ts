import {
  Component,
  ChangeDetectionStrategy,
  signal,
  inject,
  ViewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TableModule, Table } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { InputTextModule } from 'primeng/inputtext';
import { CalendarModule } from 'primeng/calendar';
import { MessageService } from 'primeng/api';
import { ToastModule } from 'primeng/toast';

import { AdminService } from '../../services/admin.service';
import { AuditLogFilter } from '../../models/admin.model';

@Component({
  selector: 'app-audit-log',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    TableModule,
    ButtonModule,
    InputTextModule,
    CalendarModule,
    ToastModule,
  ],
  providers: [MessageService],
  template: `
    <div class="p-6">
      <h2 class="text-3xl font-bold mb-6">Audit Log</h2>

      <p-toast></p-toast>

      <!-- Filters -->
      <div class="grid grid-cols-12 gap-4 mb-6 p-4 bg-white rounded-lg shadow">
        <div class="col-12 md:col-4">
          <label class="block text-sm font-medium mb-2">Action</label>
          <input
            pInputText
            type="text"
            [(ngModel)]="filterAction()"
            placeholder="e.g. role_change"
            class="w-full"
          />
        </div>

        <div class="col-12 md:col-4">
          <label class="block text-sm font-medium mb-2">From Date</label>
          <p-calendar
            [(ngModel)]="filterFrom()"
            dateFormat="yy-mm-dd"
            [showTime]="true"
            [showSeconds]="true"
          ></p-calendar>
        </div>

        <div class="col-12 md:col-4">
          <label class="block text-sm font-medium mb-2">To Date</label>
          <p-calendar
            [(ngModel)]="filterTo()"
            dateFormat="yy-mm-dd"
            [showTime]="true"
            [showSeconds]="true"
          ></p-calendar>
        </div>

        <div class="col-12">
          <button
            pButton
            type="button"
            label="Search"
            (click)="applyFilters()"
            [loading]="adminService.loading()"
            class="mr-2"
          ></button>
          <button
            pButton
            type="button"
            label="Clear"
            (click)="clearFilters()"
            class="p-button-secondary"
          ></button>
        </div>
      </div>

      <!-- Table -->
      <p-table
        #dt
        [value]="adminService.auditLog()"
        [loading]="adminService.loading()"
        [lazy]="true"
        (onLazyLoad)="onLazyLoad($event)"
        [totalRecords]="totalRecords()"
        [paginator]="true"
        [rows]="25"
        responsiveLayout="scroll"
        [tableStyle]="{ 'min-width': '50rem' }"
      >
        <ng-template pTemplate="header">
          <tr>
            <th>Timestamp</th>
            <th>User ID</th>
            <th>Action</th>
            <th>Target</th>
            <th>IP Address</th>
            <th>User Agent</th>
          </tr>
        </ng-template>

        <ng-template pTemplate="body" let-entry>
          <tr>
            <td>{{ entry.created_at | date: 'short' }}</td>
            <td>{{ entry.user_id }}</td>
            <td>{{ entry.action }}</td>
            <td>{{ entry.target_entity }} / {{ entry.target_id }}</td>
            <td>{{ entry.ip_address }}</td>
            <td class="text-xs">{{ entry.user_agent | slice: 0: 40 }}...</td>
          </tr>
        </ng-template>

        <ng-template pTemplate="emptymessage">
          <tr>
            <td colspan="6" class="text-center py-4">No audit logs found</td>
          </tr>
        </ng-template>
      </p-table>

      <!-- Export Button -->
      <div class="mt-4">
        <button
          pButton
          type="button"
          label="Export to CSV"
          icon="pi pi-download"
          (click)="exportCSV()"
          class="p-button-sm"
        ></button>
      </div>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuditLogComponent {
  readonly adminService = inject(AdminService);
  private readonly messageService = inject(MessageService);

  @ViewChild('dt') table: Table | undefined;

  readonly filterAction = signal('');
  readonly filterFrom = signal<Date | null>(null);
  readonly filterTo = signal<Date | null>(null);
  readonly totalRecords = signal(0);
  readonly currentPage = signal(1);

  applyFilters(): void {
    this.currentPage.set(1);
    this.loadAuditLog(1);
  }

  clearFilters(): void {
    this.filterAction.set('');
    this.filterFrom.set(null);
    this.filterTo.set(null);
    this.currentPage.set(1);
    this.loadAuditLog(1);
  }

  onLazyLoad(event: any): void {
    const page = (event.first || 0) / (event.rows || 25) + 1;
    this.loadAuditLog(page);
  }

  private loadAuditLog(page: number): void {
    const filter: AuditLogFilter = {
      page,
      page_size: 25,
      action: this.filterAction() || undefined,
      from: this.filterFrom() ? this.filterFrom()!.toISOString() : undefined,
      to: this.filterTo() ? this.filterTo()!.toISOString() : undefined,
    };

    this.adminService.loadAuditLog(filter).subscribe({
      next: (result) => {
        this.totalRecords.set(result.total);
        this.currentPage.set(page);
      },
      error: (err) => {
        this.messageService.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to load audit log',
        });
      },
    });
  }

  exportCSV(): void {
    if (this.table) {
      this.table.exportCSV();
    }
  }
}
