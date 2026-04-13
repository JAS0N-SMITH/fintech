import type { Page, Locator } from '@playwright/test';
import { expect } from '@playwright/test';

/**
 * Page Object Model for the Dashboard page (/dashboard).
 */
export class DashboardPage {
  readonly url = '/dashboard';

  constructor(private readonly page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto(this.url);
    await this.waitForLoad();
  }

  private async waitForLoad(): Promise<void> {
    // Wait for summary cards to render
    await this.page.waitForSelector('[data-testid="portfolio-value-card"]', {
      state: 'visible',
      timeout: 5000,
    });
  }

  // ---- Summary Cards ----

  get portfolioValueCard(): Locator {
    return this.page.locator('[data-testid="portfolio-value-card"]');
  }

  get gainLossCard(): Locator {
    return this.page.locator('[data-testid="gain-loss-card"]');
  }

  get dayChangeCard(): Locator {
    return this.page.locator('[data-testid="day-change-card"]');
  }

  async getPortfolioValue(): Promise<string> {
    return this.portfolioValueCard.locator('[data-testid="value"]').textContent();
  }

  async getGainLoss(): Promise<string> {
    return this.gainLossCard.locator('[data-testid="value"]').textContent();
  }

  // ---- Allocation Chart ----

  get allocationChart(): Locator {
    return this.page.locator('[data-testid="allocation-chart"]');
  }

  async isAllocationChartVisible(): Promise<boolean> {
    return this.allocationChart.isVisible();
  }

  // ---- Top Movers Tables ----

  get topGainersTable(): Locator {
    return this.page.locator('[data-testid="top-gainers-table"]');
  }

  get topLosersTable(): Locator {
    return this.page.locator('[data-testid="top-losers-table"]');
  }

  async getTopGainers(): Promise<string[]> {
    const rows = this.topGainersTable.locator('tbody tr');
    const count = await rows.count();
    const gainers: string[] = [];
    for (let i = 0; i < count; i++) {
      const symbol = await rows.nth(i).locator('td:first-child').textContent();
      gainers.push(symbol?.trim() || '');
    }
    return gainers;
  }

  async getTopLosers(): Promise<string[]> {
    const rows = this.topLosersTable.locator('tbody tr');
    const count = await rows.count();
    const losers: string[] = [];
    for (let i = 0; i < count; i++) {
      const symbol = await rows.nth(i).locator('td:first-child').textContent();
      losers.push(symbol?.trim() || '');
    }
    return losers;
  }

  // ---- Navigation ----

  /**
   * Click on a symbol in the top movers to navigate to the ticker detail page.
   */
  async clickOnGainer(index: number): Promise<void> {
    await this.topGainersTable.locator('tbody tr').nth(index).click();
    // Wait for navigation
    await this.page.waitForURL(/\/tickers\/[A-Z0-9.-]+/);
  }

  // ---- Accessibility ----

  /**
   * Returns all interactive elements on the dashboard for keyboard navigation testing.
   */
  get interactiveElements(): Locator {
    return this.page.locator('button, a[href], [role="button"]');
  }

  async getInteractiveCount(): Promise<number> {
    return this.interactiveElements.count();
  }
}
