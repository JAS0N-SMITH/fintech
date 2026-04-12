import type { Page, Locator } from '@playwright/test';
import { expect } from '@playwright/test';

/**
 * Page Object Model for the Admin Panel.
 */
export class AdminPage {
  readonly url = '/admin';

  constructor(private readonly page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto(this.url);
    await this.waitForLoad();
  }

  private async waitForLoad(): Promise<void> {
    // Wait for admin layout sidebar to appear
    await this.page.waitForSelector('app-admin-layout', { state: 'visible' });
  }

  // ---- Navigation ----

  async navigateToUsers(): Promise<void> {
    await this.page.getByRole('menuitem', { name: /users/i }).click();
    await this.page.waitForURL('**/admin/users');
    await this.page.waitForSelector('p-table', { state: 'visible' });
  }

  async navigateToAuditLog(): Promise<void> {
    await this.page.getByRole('menuitem', { name: /audit log/i }).click();
    await this.page.waitForURL('**/admin/audit-log');
    await this.page.waitForSelector('p-table', { state: 'visible' });
  }

  async navigateToHealth(): Promise<void> {
    await this.page.getByRole('menuitem', { name: /system health/i }).click();
    await this.page.waitForURL('**/admin/health');
    await this.page.waitForSelector('p-card', { state: 'visible' });
  }

  async backToApp(): Promise<void> {
    await this.page.getByRole('menuitem', { name: /back to app/i }).click();
    await this.page.waitForURL('/');
  }

  // ---- User Management ----

  async getUserRow(email: string): Promise<Locator> {
    return this.page.locator('p-table tbody tr').filter({ hasText: email });
  }

  async openRoleDialog(email: string): Promise<void> {
    const row = await this.getUserRow(email);
    const editButton = row.locator('button[icon="pi pi-pencil"]').first();
    await editButton.click();
    await this.page.waitForSelector('p-dialog', { state: 'visible' });
  }

  async selectNewRole(role: 'user' | 'admin'): Promise<void> {
    const dropdown = this.page.locator('p-dropdown');
    await dropdown.click();
    const option = this.page.getByRole('option', { name: new RegExp(role, 'i') });
    await option.click();
  }

  async confirmRoleChange(): Promise<void> {
    const dialog = this.page.locator('p-dialog');
    const updateButton = dialog.getByRole('button', { name: /update/i });
    await updateButton.click();
    await this.page.waitForSelector('p-dialog', { state: 'hidden' });
  }

  async expectSuccessToast(message: string): Promise<void> {
    const toast = this.page.locator('p-toast .p-toast-message.ng-trigger-messageAnimation');
    await expect(toast).toContainText(message);
  }

  // ---- Audit Log ----

  async filterByAction(action: string): Promise<void> {
    const actionInput = this.page.locator('input[placeholder*="e.g. role_change"]');
    await actionInput.fill(action);
  }

  async clickSearch(): Promise<void> {
    await this.page.getByRole('button', { name: /search/i }).click();
    await this.page.waitForSelector('p-table tbody', { state: 'visible' });
  }

  async getAuditLogEntries(): Promise<string[]> {
    const rows = this.page.locator('p-table tbody tr');
    const count = await rows.count();
    const entries: string[] = [];
    for (let i = 0; i < count; i++) {
      const text = await rows.nth(i).textContent();
      entries.push(text || '');
    }
    return entries;
  }

  // ---- Health Status ----

  async getHealthStatus(component: 'db' | 'finnhub_api' | 'websocket'): Promise<string> {
    const card = this.page.locator('p-card').filter({
      has: this.page.getByRole('heading', {
        name: new RegExp(
          component === 'db'
            ? 'Database'
            : component === 'finnhub_api'
              ? 'Finnhub'
              : 'WebSocket',
          'i'
        ),
      }),
    });
    const tag = card.locator('p-tag');
    return await tag.getAttribute('ng-reflect-value');
  }
}
