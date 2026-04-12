import { Page, Locator } from '@playwright/test';

/**
 * WatchlistPage encapsulates interactions with the watchlist pages.
 * Follows the Page Object Model pattern for maintainability and reusability.
 */
export class WatchlistPage {
  readonly page: Page;

  // List page selectors
  readonly newWatchlistButton: Locator;
  readonly watchlistTable: Locator;
  readonly watchlistRows: Locator;
  readonly createDialog: Locator;
  readonly watchlistNameInput: Locator;
  readonly dialogSaveButton: Locator;
  readonly dialogCancelButton: Locator;
  readonly deleteButtons: Locator;
  readonly editButtons: Locator;

  // Detail page selectors
  readonly backButton: Locator;
  readonly addTickerButton: Locator;
  readonly itemsTable: Locator;
  readonly itemRows: Locator;
  readonly tickerSearchInput: Locator;
  readonly tickerSearchAddButton: Locator;
  readonly removeItemButtons: Locator;
  readonly confirmDialog: Locator;
  readonly confirmDeleteButton: Locator;

  constructor(page: Page) {
    this.page = page;

    // List page selectors
    this.newWatchlistButton = page.getByRole('button', { name: /new watchlist/i });
    this.watchlistTable = page.getByRole('table', { name: /watchlists table/i });
    this.watchlistRows = page.locator('p-table tbody tr');
    this.createDialog = page.getByRole('dialog');
    this.watchlistNameInput = page.locator('#watchlist-name');
    this.dialogSaveButton = page.getByRole('button', { name: /save/i }).last();
    this.dialogCancelButton = page.getByRole('button', { name: /cancel/i }).last();
    this.deleteButtons = page.getByRole('button', { name: /delete/i });
    this.editButtons = page.getByRole('button', { name: /rename/i });

    // Detail page selectors
    this.backButton = page.locator('button i.pi-arrow-left').first();
    this.addTickerButton = page.getByRole('button', { name: /add ticker/i });
    this.itemsTable = page.getByRole('table', { name: /watchlist items table/i });
    this.itemRows = page.locator('p-table tbody tr');
    this.tickerSearchInput = page.locator('#ticker-input');
    this.tickerSearchAddButton = page.getByRole('button', { name: /add/i }).last();
    this.removeItemButtons = page.getByRole('button', { name: /remove/i });
    this.confirmDialog = page.getByRole('dialog');
    this.confirmDeleteButton = page.getByRole('button', { name: /confirm/i });
  }

  // List page methods

  async goto(): Promise<void> {
    await this.page.goto('/watchlists');
    await this.page.waitForLoadState('networkidle');
  }

  async createWatchlist(name: string): Promise<void> {
    await this.newWatchlistButton.click();
    await this.createDialog.waitFor({ state: 'visible' });
    await this.watchlistNameInput.fill(name);
    await this.dialogSaveButton.click();
    await this.createDialog.waitFor({ state: 'hidden' });
  }

  async getWatchlistCount(): Promise<number> {
    return await this.watchlistRows.count();
  }

  async openWatchlist(name: string): Promise<void> {
    await this.page.getByRole('button', { name: `View ${name}` }).click();
    await this.page.waitForLoadState('networkidle');
  }

  async renameWatchlist(oldName: string, newName: string): Promise<void> {
    // Find the row with oldName and click edit button
    const row = this.watchlistRows.filter({
      has: this.page.getByText(oldName),
    });
    const editButton = row.locator('button[aria-label*="Rename"]');
    await editButton.click();
    await this.createDialog.waitFor({ state: 'visible' });
    await this.watchlistNameInput.clear();
    await this.watchlistNameInput.fill(newName);
    await this.dialogSaveButton.click();
    await this.createDialog.waitFor({ state: 'hidden' });
  }

  async deleteWatchlist(name: string): Promise<void> {
    const row = this.watchlistRows.filter({
      has: this.page.getByText(name),
    });
    const deleteButton = row.locator('button[aria-label*="Delete"]');
    await deleteButton.click();
    await this.confirmDialog.waitFor({ state: 'visible' });
    await this.confirmDeleteButton.click();
    await this.confirmDialog.waitFor({ state: 'hidden' });
  }

  // Detail page methods

  async addTicker(symbol: string): Promise<void> {
    await this.addTickerButton.click();
    await this.createDialog.waitFor({ state: 'visible' });
    await this.tickerSearchInput.fill(symbol);
    // Wait for autocomplete suggestions
    await this.page.waitForTimeout(300);
    await this.page
      .getByRole('option', { name: new RegExp(symbol, 'i') })
      .first()
      .click();
    await this.tickerSearchAddButton.click();
    await this.createDialog.waitFor({ state: 'hidden' });
  }

  async getItemCount(): Promise<number> {
    return await this.itemRows.count();
  }

  async getItemBySymbol(symbol: string): Promise<Locator> {
    return this.itemRows.filter({
      has: this.page.getByText(symbol),
    });
  }

  async getItemPrice(symbol: string): Promise<string> {
    const row = await this.getItemBySymbol(symbol);
    const priceCell = row.locator('td').nth(1);
    return await priceCell.textContent();
  }

  async removeTicker(symbol: string): Promise<void> {
    const row = await this.getItemBySymbol(symbol);
    const removeButton = row.locator('button[aria-label*="Remove"]');
    await removeButton.click();
    await this.confirmDialog.waitFor({ state: 'visible' });
    await this.confirmDeleteButton.click();
    await this.confirmDialog.waitFor({ state: 'hidden' });
  }

  async goBack(): Promise<void> {
    await this.backButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  // Utility methods

  async waitForPriceUpdate(symbol: string): Promise<void> {
    // Wait for price to update from WebSocket
    const row = await this.getItemBySymbol(symbol);
    const priceCell = row.locator('td').nth(1);
    await priceCell.waitFor({ state: 'visible' });
    // Give WebSocket a moment to update
    await this.page.waitForTimeout(500);
  }
}
