import type { Page, Locator } from '@playwright/test';
import { expect } from '@playwright/test';

/**
 * Page Object Model for the Portfolio List page (/portfolios).
 */
export class PortfolioListPage {
  readonly url = '/portfolios';

  constructor(private readonly page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto(this.url);
    await this.waitForLoad();
  }

  private async waitForLoad(): Promise<void> {
    // Table renders once loading spinner is gone
    await this.page.waitForSelector('p-table', { state: 'visible' });
  }

  // ---- Header actions ----

  get newPortfolioButton(): Locator {
    return this.page.getByRole('button', { name: /new portfolio/i });
  }

  async clickNewPortfolio(): Promise<void> {
    await this.newPortfolioButton.click();
    await this.page.waitForSelector('p-dialog', { state: 'visible' });
  }

  // ---- Dialog (create / edit) ----

  async fillPortfolioForm(name: string, description?: string): Promise<void> {
    const dialog = this.page.locator('p-dialog');
    await dialog.locator('#pf-name').fill(name);
    if (description) {
      await dialog.locator('#pf-desc').fill(description);
    }
  }

  async submitPortfolioForm(): Promise<void> {
    const dialog = this.page.locator('p-dialog');
    await dialog.getByRole('button', { name: /create portfolio/i }).click();
    // Dialog closes on success
    await this.page.waitForSelector('p-dialog', { state: 'hidden' });
  }

  // ---- Table ----

  /**
   * Returns a row locator for a portfolio with the given name.
   * Throws if not found.
   */
  rowFor(name: string): Locator {
    return this.page.locator('p-table tbody tr').filter({ hasText: name });
  }

  async portfolioNames(): Promise<string[]> {
    const cells = await this.page
      .locator('p-table tbody tr td:first-child button')
      .allTextContents();
    return cells.map((t) => t.trim());
  }

  async clickPortfolioName(name: string): Promise<void> {
    await this.page
      .locator('p-table tbody tr td:first-child button')
      .filter({ hasText: name })
      .click();
  }

  async clickEditFor(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.getByRole('button', { name: /edit/i }).click();
    await this.page.waitForSelector('p-dialog', { state: 'visible' });
  }

  async clickDeleteFor(name: string): Promise<void> {
    const row = this.rowFor(name);
    await row.getByRole('button', { name: /delete/i }).click();
    // Confirmation dialog appears
    await this.page.waitForSelector('p-confirmdialog', { state: 'visible' });
  }

  async confirmDelete(): Promise<void> {
    await this.page.getByRole('button', { name: /yes/i }).click();
    await this.page.waitForSelector('p-confirmdialog', { state: 'hidden' });
  }

  // ---- Assertions ----

  async expectPortfolioVisible(name: string): Promise<void> {
    await expect(this.rowFor(name)).toBeVisible();
  }

  async expectPortfolioNotVisible(name: string): Promise<void> {
    await expect(
      this.page
        .locator('p-table tbody tr td:first-child button')
        .filter({ hasText: name }),
    ).toHaveCount(0);
  }
}
