import {
  Component,
  ChangeDetectionStrategy,
  signal,
  inject,
  OnInit,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { DropdownModule } from 'primeng/dropdown';
import { ToastModule } from 'primeng/toast';
import { MessageService } from 'primeng/api';
import { TooltipModule } from 'primeng/tooltip';

import { AdminService } from '../../services/admin.service';
import { AdminUser } from '../../models/admin.model';

@Component({
  selector: 'app-user-management',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    TableModule,
    ButtonModule,
    DialogModule,
    DropdownModule,
    ToastModule,
    TooltipModule,
  ],
  providers: [MessageService],
  template: `
    <div class="p-6">
      <h2 class="text-3xl font-bold mb-6">User Management</h2>

      <p-toast></p-toast>

      <p-table
        [value]="adminService.users()"
        [loading]="adminService.loading()"
        responsiveLayout="scroll"
        [paginator]="true"
        [rows]="25"
        [totalRecords]="totalUsers"
      >
        <ng-template pTemplate="header">
          <tr>
            <th>Email</th>
            <th>Display Name</th>
            <th>Role</th>
            <th>Created At</th>
            <th>Actions</th>
          </tr>
        </ng-template>

        <ng-template pTemplate="body" let-user>
          <tr>
            <td>{{ user.email }}</td>
            <td>{{ user.display_name }}</td>
            <td>
              <span
                [class]="
                  user.role === 'admin'
                    ? 'bg-red-100 text-red-800 px-2 py-1 rounded'
                    : 'bg-blue-100 text-blue-800 px-2 py-1 rounded'
                "
              >
                {{ user.role }}
              </span>
            </td>
            <td>{{ user.created_at | date: 'short' }}</td>
            <td>
              <button
                pButton
                type="button"
                icon="pi pi-pencil"
                class="p-button-sm p-button-rounded mr-2"
                (click)="openRoleDialog(user)"
              ></button>
              <button
                pButton
                type="button"
                icon="pi pi-lock"
                class="p-button-sm p-button-rounded p-button-secondary"
                pTooltip="Coming soon"
                tooltipPosition="top"
              ></button>
            </td>
          </tr>
        </ng-template>
      </p-table>
    </div>

    <!-- Role Change Dialog -->
    <p-dialog
      [(visible)]="showRoleDialog()"
      [header]="'Change Role: ' + selectedUser()?.email"
      [modal]="true"
      [style]="{ width: '50vw' }"
    >
      <div class="grid grid-cols-12 gap-4 mb-4">
        <label class="col-12 font-bold">New Role</label>
        <p-dropdown
          class="col-12"
          [(ngModel)]="newRole()"
          [options]="roleOptions"
          optionLabel="label"
          optionValue="value"
        ></p-dropdown>
      </div>

      <ng-template pTemplate="footer">
        <button
          pButton
          type="button"
          label="Cancel"
          icon="pi pi-times"
          (click)="showRoleDialog.set(false)"
          class="p-button-text"
        ></button>
        <button
          pButton
          type="button"
          label="Update"
          icon="pi pi-check"
          (click)="confirmRoleChange()"
          [loading]="adminService.loading()"
        ></button>
      </ng-template>
    </p-dialog>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UserManagementComponent implements OnInit {
  readonly adminService = inject(AdminService);
  private readonly messageService = inject(MessageService);

  readonly showRoleDialog = signal(false);
  readonly selectedUser = signal<AdminUser | null>(null);
  readonly newRole = signal<'user' | 'admin'>('user');
  readonly totalUsers = signal(0);

  readonly roleOptions = [
    { label: 'User', value: 'user' as const },
    { label: 'Admin', value: 'admin' as const },
  ];

  ngOnInit(): void {
    this.loadUsers();
  }

  loadUsers(): void {
    this.adminService.loadUsers(1, 25).subscribe({
      next: (result) => {
        this.totalUsers.set(result.total);
      },
      error: (err) => {
        this.messageService.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to load users',
        });
      },
    });
  }

  openRoleDialog(user: AdminUser): void {
    this.selectedUser.set(user);
    this.newRole.set(user.role);
    this.showRoleDialog.set(true);
  }

  confirmRoleChange(): void {
    const user = this.selectedUser();
    if (!user) return;

    this.adminService.patchRole(user.id, this.newRole()).subscribe({
      next: () => {
        this.messageService.add({
          severity: 'success',
          summary: 'Success',
          detail: `User role updated to ${this.newRole()}`,
        });
        this.showRoleDialog.set(false);
      },
      error: (err) => {
        this.messageService.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to update user role',
        });
      },
    });
  }
}
