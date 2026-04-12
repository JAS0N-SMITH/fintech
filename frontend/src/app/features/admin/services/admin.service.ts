import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { tap } from 'rxjs/operators';
import { inject } from '@angular/core';

import {
  AdminUser,
  AdminUserList,
  AuditLogEntry,
  AuditLogList,
  AuditLogFilter,
  HealthStatus,
  PatchRoleInput,
} from '../models/admin.model';

@Injectable({ providedIn: 'root' })
export class AdminService {
  private readonly http = inject(HttpClient);
  private readonly apiUrl = '/api/v1/admin';

  // Signal state
  private readonly _users = signal<AdminUser[]>([]);
  private readonly _auditLog = signal<AuditLogEntry[]>([]);
  private readonly _health = signal<HealthStatus | null>(null);
  private readonly _loading = signal(false);
  private readonly _error = signal<string | null>(null);

  // Public read-only signals
  readonly users = this._users.asReadonly();
  readonly auditLog = this._auditLog.asReadonly();
  readonly health = this._health.asReadonly();
  readonly loading = this._loading.asReadonly();
  readonly error = this._error.asReadonly();

  /**
   * Load a paginated list of users.
   */
  loadUsers(page: number = 1, pageSize: number = 25): Observable<AdminUserList> {
    this._loading.set(true);
    this._error.set(null);

    return this.http
      .get<AdminUserList>(`${this.apiUrl}/users`, {
        params: { page: page.toString(), page_size: pageSize.toString() },
      })
      .pipe(
        tap((result) => {
          this._users.set(result.users);
          this._loading.set(false);
        })
      );
  }

  /**
   * Change a user's role.
   */
  patchRole(userId: string, role: 'user' | 'admin'): Observable<AdminUser> {
    this._loading.set(true);
    this._error.set(null);

    const body: PatchRoleInput = { role };

    return this.http
      .patch<AdminUser>(`${this.apiUrl}/users/${userId}/role`, body)
      .pipe(
        tap((user) => {
          // Update user in the list
          const users = this._users();
          const idx = users.findIndex((u) => u.id === user.id);
          if (idx >= 0) {
            users[idx] = user;
            this._users.set([...users]);
          }
          this._loading.set(false);
        })
      );
  }

  /**
   * Load a paginated list of audit log entries.
   */
  loadAuditLog(filter: AuditLogFilter): Observable<AuditLogList> {
    this._loading.set(true);
    this._error.set(null);

    const params: Record<string, string> = {
      page: filter.page.toString(),
      page_size: filter.page_size.toString(),
    };

    if (filter.user_id) {
      params['user_id'] = filter.user_id;
    }
    if (filter.action) {
      params['action'] = filter.action;
    }
    if (filter.from) {
      params['from'] = filter.from;
    }
    if (filter.to) {
      params['to'] = filter.to;
    }

    return this.http
      .get<AuditLogList>(`${this.apiUrl}/audit-log`, { params })
      .pipe(
        tap((result) => {
          this._auditLog.set(result.entries);
          this._loading.set(false);
        })
      );
  }

  /**
   * Load system health status.
   */
  loadHealth(): Observable<HealthStatus> {
    this._loading.set(true);
    this._error.set(null);

    return this.http.get<HealthStatus>(`${this.apiUrl}/health`).pipe(
      tap((status) => {
        this._health.set(status);
        this._loading.set(false);
      })
    );
  }
}
