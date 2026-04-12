import type { Page, Locator } from '@playwright/test';
import { expect } from '@playwright/test';

/**
 * Page Object Model for the Ticker Detail page (/tickers/:symbol).
 */
export class TickerDetailPage {
  constructor(private readonly page: Page) {}

  // ---- Navigation ----

  async goto(symbol: string): Promise<void> {
    await this.page.goto(`/tickers/${symbol}`);
    // Wait for the page title to render
    await this.page.waitForSelector('h1', { state: 'visible' });
  }

  async clickBackButton(): Promise<void> {
    await this.page.getByRole('button', { name: /go back to previous page/i }).click();
  }

  // ---- Header ----

  heading(): Locator {
    return this.page.locator('h1');
  }

  async getSymbol(): Promise<string> {
    return this.heading().textContent() || '';
  }

  // ---- Price display ----

  async getCurrentPrice(): Promise<string | null> {
    const priceEl = this.page.locator('h1 + p');
    if (!(await priceEl.isVisible())) return null;
    return priceEl.textContent();
  }

  // ---- Connection badge ----

  async getConnectionStatus(): Promise<string> {
    const tag = this.page.locator('p-tag').first();
    return tag.textContent() || '';
  }

  // ---- Time range selector ----

  async selectTimeframe(timeframe: '1D' | '1W' | '1M' | '3M' | '1Y' | 'ALL'): Promise<void> {
    const button = this.page.getByRole('button', { name: new RegExp(`^${timeframe}$`) });
    await button.click();
    // Wait for chart to update (barsLoading becomes false)
    await this.page.waitForTimeout(500);
  }

  async getSelectedTimeframe(): Promise<string> {
    // Find the button with filled/primary styling (p-button with severity="primary")
    const buttons = this.page.locator('p-button');
    for (let i = 0; i < (await buttons.count()); i++) {
      const button = buttons.nth(i);
      const severity = await button.getAttribute('severity');
      if (severity === 'primary') {
        return (await button.getAttribute('label')) || '';
      }
    }
    return '';
  }

  // ---- Chart section ----

  async waitForChartVisible(): Promise<void> {
    // Wait for the Lightweight Charts container
    await this.page.waitForSelector('app-ticker-chart', { state: 'visible' });
    // Wait for the canvas element inside the chart
    await this.page.locator('app-ticker-chart div').first().waitFor({ state: 'visible' });
  }

  async isChartLoading(): Promise<boolean> {
    // Loading skeleton div has animate-pulse class
    const skeleton = this.page.locator('app-ticker-chart ~ div.animate-pulse').first();
    return skeleton.isVisible();
  }

  // ---- Key stats card ----

  async waitForKeyStatsVisible(): Promise<void> {
    await this.page.locator('app-key-stats-card').waitFor({ state: 'visible' });
  }

  async getKeyStatValue(label: string): Promise<string | null> {
    // Find the stat by its label, then get the next value element
    const labelEl = this.page.locator('text=' + label).first();
    if (!(await labelEl.isVisible())) return null;
    const statDiv = labelEl.locator('..');
    const valueEl = statDiv.locator('p, span').nth(1); // Skip the label, get value
    return valueEl.textContent();
  }

  // ---- Position summary card ----

  async waitForPositionSummaryVisible(): Promise<void> {
    await this.page.locator('app-position-summary-card').waitFor({ state: 'visible' });
  }

  async getPositionValue(label: string): Promise<string | null> {
    const labelEl = this.page.locator('text=' + label).first();
    if (!(await labelEl.isVisible())) return null;
    const statDiv = labelEl.locator('..');
    const valueEl = statDiv.locator('p, span').nth(1);
    return valueEl.textContent();
  }

  async hasPosition(): Promise<boolean> {
    const card = this.page.locator('app-position-summary-card');
    const emptyMsg = this.page.locator('text=No position in this ticker');
    return (await card.isVisible()) && !(await emptyMsg.isVisible());
  }

  // ---- Transaction history ----

  async waitForTransactionTableVisible(): Promise<void> {
    await this.page.locator('app-ticker-transactions-table p-table').waitFor({ state: 'visible' });
  }

  async getTransactionCount(): Promise<number> {
    const rows = this.page.locator('app-ticker-transactions-table p-table tbody tr');
    return rows.count();
  }

  async hasTransactions(): Promise<boolean> {
    const table = this.page.locator('app-ticker-transactions-table p-table');
    const emptyMsg = this.page.locator('text=No transactions for this symbol');
    return (await table.isVisible()) && !(await emptyMsg.isVisible());
  }

  // ---- Navigation flow ----

  /**
   * Navigate from holdings table to ticker detail by clicking the symbol link.
   * Assumes you're on a page with app-holdings-table or similar symbol links.
   */
  async clickSymbolLink(symbol: string): Promise<void> {
    // Find link with the exact symbol text
    const link = this.page.locator(`a:has-text("${symbol}")`).first();
    await link.click();
    // Wait for the ticker detail page to load
    await this.waitForLoad();
  }

  /**
   * Wait for the ticker detail page to be fully loaded (chart, stats, etc.).
   */
  async waitForLoad(): Promise<void> {
    await this.page.waitForSelector('h1', { state: 'visible' });
    await this.waitForChartVisible();
    await this.waitForKeyStatsVisible();
  }
}
