import {
  Component,
  ChangeDetectionStrategy,
  signal,
  inject,
} from '@angular/core';
import { RouterLink, RouterOutlet } from '@angular/router';
import { MenuItem } from 'primeng/api';
import { MenuModule } from 'primeng/menu';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-admin-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, MenuModule],
  template: `
    <div class="flex h-screen bg-gray-100">
      <!-- Admin Sidebar -->
      <aside class="w-64 bg-white shadow-lg">
        <div class="p-6 border-b">
          <h1 class="text-2xl font-bold text-gray-800">Admin</h1>
        </div>

        <p-menu [model]="navItems()" class="mt-4" />
      </aside>

      <!-- Main Content -->
      <main class="flex-1 overflow-auto">
        <router-outlet></router-outlet>
      </main>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminLayoutComponent {
  readonly navItems = signal<MenuItem[]>([
    {
      label: 'Users',
      icon: 'pi pi-fw pi-users',
      routerLink: ['/admin/users'],
    },
    {
      label: 'Audit Log',
      icon: 'pi pi-fw pi-list',
      routerLink: ['/admin/audit-log'],
    },
    {
      label: 'System Health',
      icon: 'pi pi-fw pi-heart',
      routerLink: ['/admin/health'],
    },
    {
      separator: true,
    },
    {
      label: 'Back to App',
      icon: 'pi pi-fw pi-arrow-left',
      routerLink: ['/'],
    },
  ]);
}
