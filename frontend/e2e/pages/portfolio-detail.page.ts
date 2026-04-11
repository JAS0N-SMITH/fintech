import type { Page, Locator } from '@playwright/test';
import { expect } from '@playwright/test';

type TransactionType = 'Buy' | 'Sell' | 'Dividend' | 'Reinvested dividend';

/**
 * Page Object Model for the Portfolio Detail page (/portfolios/:id).
 */
export class PortfolioDetailPage {
  constructor(private readonly page: Page) {}

  // ---- Navigation ----

  async waitForLoad(): Promise<void> {
    // The holdings tab is active by default
    await this.page.waitForSelector('app-holdings-table', { state: 'visible' });
  }

  async clickBackButton(): Promise<void> {
    await this.page.getByRole('button', { name: /back to portfolios/i }).click();
  }

  // ---- Header ----

  heading(): Locator {
    return this.page.locator('h1');
  }

  // ---- Tabs ----

  async switchToHoldingsTab(): Promise<void> {
    await this.page.getByRole('tab', { name: /holdings/i }).click();
    await this.page.waitForSelector('app-holdings-table', { state: 'visible' });
  }

  async switchToTransactionsTab(): Promise<void> {
    await this.page.getByRole('tab', { name: /transactions/i }).click();
    // Wait for the transaction table to be visible
    await this.page.waitForSelector('p-table', { state: 'visible' });
  }

  // ---- Add transaction dialog ----

  async clickAddTransaction(): Promise<void> {
    await this.page.getByRole('button', { name: /add transaction/i }).click();
    await this.page.waitForSelector('p-dialog', { state: 'visible' });
  }

  async fillTransactionForm(opts: {
    type: TransactionType;
    symbol: string;
    date?: string; // YYYY-MM-DD, defaults to today
    quantity?: number;
    pricePerShare?: number;
    dividendPerShare?: number;
    totalAmount: number;
    notes?: string;
  }): Promise<void> {
    const dialog = this.page.locator('p-dialog');

    // Transaction type
    await dialog.locator('p-select[formcontrolname="transaction_type"]').click();
    await this.page.getByRole('option', { name: opts.type }).click();

    // Symbol
    await dialog.locator('#tx-symbol').fill(opts.symbol);
    await dialog.locator('#tx-symbol').blur(); // triggers uppercase normalization

    // Date
    const dateStr = opts.date ?? new Date().toISOString().split('T')[0];
    await dialog.locator('p-datepicker input').fill(dateStr);
    await dialog.locator('p-datepicker input').press('Tab');

    // Quantity (if visible)
    if (opts.quantity !== undefined) {
      await dialog.locator('#tx-qty input').fill(opts.quantity.toString());
    }

    // Price per share (if visible)
    if (opts.pricePerShare !== undefined) {
      await dialog.locator('#tx-price input').fill(opts.pricePerShare.toString());
    }

    // Dividend per share (if visible)
    if (opts.dividendPerShare !== undefined) {
      await dialog.locator('#tx-dps input').fill(opts.dividendPerShare.toString());
    }

    // Total amount
    await dialog.locator('#tx-total input').fill(opts.totalAmount.toString());

    // Notes
    if (opts.notes) {
      await dialog.locator('#tx-notes').fill(opts.notes);
    }
  }

  async submitTransaction(): Promise<void> {
    const dialog = this.page.locator('p-dialog');
    await dialog.getByRole('button', { name: /record transaction/i }).click();
    await this.page.waitForSelector('p-dialog', { state: 'hidden' });
  }

  // ---- Holdings table ----

  holdingRow(symbol: string): Locator {
    return this.page
      .locator('app-holdings-table tbody tr')
      .filter({ hasText: symbol });
  }

  async holdingQuantity(symbol: string): Promise<string> {
    const row = this.holdingRow(symbol);
    const cell = row.locator('td').nth(1);
    return (await cell.textContent())?.trim() ?? '';
  }

  async expectHoldingVisible(symbol: string): Promise<void> {
    await expect(this.holdingRow(symbol)).toBeVisible();
  }

  async expectHoldingNotVisible(symbol: string): Promise<void> {
    await expect(
      this.page.locator('app-holdings-table tbody tr').filter({ hasText: symbol }),
    ).toHaveCount(0);
  }

  // ---- Transaction table ----

  transactionRows(): Locator {
    // Transactions tab must be active
    return this.page.locator('p-tabpanel[value="transactions"] p-table tbody tr');
  }

  async transactionCount(): Promise<number> {
    return this.transactionRows().count();
  }

  async deleteFirstTransaction(): Promise<void> {
    await this.transactionRows().first().getByRole('button', { name: /delete/i }).click();
    await this.page.waitForSelector('p-confirmdialog', { state: 'visible' });
    await this.page.getByRole('button', { name: /yes/i }).click();
    await this.page.waitForSelector('p-confirmdialog', { state: 'hidden' });
  }
}
