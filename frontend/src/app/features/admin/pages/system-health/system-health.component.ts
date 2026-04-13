import {
  Component,
  ChangeDetectionStrategy,
  signal,
  inject,
  OnInit,
  DestroyRef,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { timer } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';
import { ToastModule } from 'primeng/toast';
import { MessageService } from 'primeng/api';

import { AdminService } from '../../services/admin.service';
import { HealthStatus } from '../../models/admin.model';

@Component({
  selector: 'app-system-health',
  standalone: true,
  imports: [CommonModule, CardModule, TagModule, ToastModule],
  providers: [MessageService],
  template: `
    <div class="p-6">
      <h2 class="text-3xl font-bold mb-6">System Health</h2>

      <p-toast></p-toast>

      <div class="grid grid-cols-12 gap-6" *ngIf="adminService.health() as health">
        <!-- Database Health -->
        <div class="col-12 md:col-6 lg:col-4">
          <p-card>
            <ng-template pTemplate="header">
              <div class="text-center py-4 bg-gray-100">
                <i class="pi pi-database text-3xl"></i>
              </div>
            </ng-template>

            <h3 class="text-xl font-bold mb-4">Database</h3>

            <p-tag
              [value]="health.db"
              [severity]="getStatusSeverity(health.db)"
              class="w-full text-center py-2"
            ></p-tag>

            <p class="text-sm text-gray-500 mt-4 text-center">
              {{ formatTime(health.timestamp) }}
            </p>
          </p-card>
        </div>

        <!-- Finnhub API Health -->
        <div class="col-12 md:col-6 lg:col-4">
          <p-card>
            <ng-template pTemplate="header">
              <div class="text-center py-4 bg-gray-100">
                <i class="pi pi-chart-line text-3xl"></i>
              </div>
            </ng-template>

            <h3 class="text-xl font-bold mb-4">Finnhub API</h3>

            <p-tag
              [value]="health.finnhub_api"
              [severity]="getStatusSeverity(health.finnhub_api)"
              class="w-full text-center py-2"
            ></p-tag>

            <p class="text-sm text-gray-500 mt-4 text-center">
              {{ formatTime(health.timestamp) }}
            </p>
          </p-card>
        </div>

        <!-- WebSocket Connections -->
        <div class="col-12 md:col-6 lg:col-4">
          <p-card>
            <ng-template pTemplate="header">
              <div class="text-center py-4 bg-gray-100">
                <i class="pi pi-sitemap text-3xl"></i>
              </div>
            </ng-template>

            <h3 class="text-xl font-bold mb-4">WebSocket Connections</h3>

            <div
              class="text-center text-3xl font-bold text-blue-600 py-4 bg-blue-50 rounded"
            >
              {{ health.websocket_count }}
            </div>

            <p class="text-sm text-gray-500 mt-4 text-center">
              {{ formatTime(health.timestamp) }}
            </p>
          </p-card>
        </div>
      </div>

      <div *ngIf="!adminService.health()" class="text-center py-8">
        <p class="text-gray-500">Loading health status...</p>
      </div>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SystemHealthComponent implements OnInit {
  readonly adminService = inject(AdminService);
  private readonly messageService = inject(MessageService);
  private readonly destroyRef = inject(DestroyRef);

  ngOnInit(): void {
    // Load health status immediately
    this.loadHealth();

    // Poll every 30 seconds
    timer(0, 30000)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => {
        this.loadHealth();
      });
  }

  private loadHealth(): void {
    this.adminService.loadHealth().subscribe({
      error: (err) => {
        this.messageService.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to load system health',
        });
      },
    });
  }

  getStatusSeverity(
    status: 'healthy' | 'unhealthy' | 'unavailable'
  ): 'success' | 'warn' | 'danger' | 'info' {
    switch (status) {
      case 'healthy':
        return 'success';
      case 'unhealthy':
        return 'warn';
      case 'unavailable':
        return 'danger';
      default:
        return 'info';
    }
  }

  formatTime(timestamp: string): string {
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  }
}
