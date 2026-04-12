import { Routes } from '@angular/router';
import { AdminLayoutComponent } from './admin-layout/admin-layout.component';

export const ADMIN_ROUTES: Routes = [
  {
    path: '',
    component: AdminLayoutComponent,
    children: [
      {
        path: '',
        redirectTo: 'users',
        pathMatch: 'full',
      },
      {
        path: 'users',
        loadComponent: () =>
          import('./pages/user-management/user-management.component').then(
            (m) => m.UserManagementComponent
          ),
      },
      {
        path: 'audit-log',
        loadComponent: () =>
          import('./pages/audit-log/audit-log.component').then(
            (m) => m.AuditLogComponent
          ),
      },
      {
        path: 'health',
        loadComponent: () =>
          import('./pages/system-health/system-health.component').then(
            (m) => m.SystemHealthComponent
          ),
      },
    ],
  },
];
