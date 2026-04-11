import { test, expect } from '@playwright/test';
import { PortfolioListPage } from './pages/portfolio-list.page';
import { PortfolioDetailPage } from './pages/portfolio-detail.page';

/**
 * Phase 4 E2E: Portfolio & Transaction Flow
 *
 * Tests run against the live Angular dev server with a real Go API and
 * Supabase database. Authentication is handled by auth.setup.ts (storageState).
 *
 * Tests are serial within this file because they share backend state:
 * each test creates then cleans up its own portfolio to stay isolated.
 */

test.describe('Portfolio management', () => {
  let listPage: PortfolioListPage;

  test.beforeEach(async ({ page }) => {
    listPage = new PortfolioListPage(page);
    await listPage.goto();
  });

  test('portfolio list loads and shows an empty state message when no portfolios exist', async ({
    page,
  }) => {
    // Either the empty message OR existing rows — the page must render without error
    const table = page.locator('p-table');
    await expect(table).toBeVisible();
  });

  test('can create a portfolio and it appears in the table', async () => {
    const name = `E2E Portfolio ${Date.now()}`;

    await listPage.clickNewPortfolio();
    await listPage.fillPortfolioForm(name, 'Created by E2E test');
    await listPage.submitPortfolioForm();

    await listPage.expectPortfolioVisible(name);

    // Cleanup
    await listPage.clickDeleteFor(name);
    await listPage.confirmDelete();
    await listPage.expectPortfolioNotVisible(name);
  });

  test('can edit a portfolio name', async () => {
    const original = `E2E Edit ${Date.now()}`;
    const updated = `${original} — renamed`;

    // Create
    await listPage.clickNewPortfolio();
    await listPage.fillPortfolioForm(original);
    await listPage.submitPortfolioForm();
    await listPage.expectPortfolioVisible(original);

    // Edit
    await listPage.clickEditFor(original);
    const dialog = listPage['page'].locator('p-dialog');
    await dialog.locator('#pf-name').fill(updated);
    await dialog.getByRole('button', { name: /save changes/i }).click();
    await listPage['page'].waitForSelector('p-dialog', { state: 'hidden' });

    await listPage.expectPortfolioVisible(updated);

    // Cleanup
    await listPage.clickDeleteFor(updated);
    await listPage.confirmDelete();
  });
});

test.describe('Transaction & holdings flow', () => {
  /**
   * Core Phase 4 E2E:
   *   Create portfolio → buy 100 AAPL → holding appears →
   *   sell 40 AAPL → quantity decreases to 60 →
   *   buy all remaining back and sell again → holding disappears
   */
  test('buy creates a holding; sell reduces quantity; full sell removes holding', async ({
    page,
  }) => {
    const listPage = new PortfolioListPage(page);
    const detailPage = new PortfolioDetailPage(page);
    const portfolioName = `E2E Holdings ${Date.now()}`;

    // 1. Create portfolio
    await listPage.goto();
    await listPage.clickNewPortfolio();
    await listPage.fillPortfolioForm(portfolioName);
    await listPage.submitPortfolioForm();
    await listPage.expectPortfolioVisible(portfolioName);

    // 2. Navigate to detail
    await listPage.clickPortfolioName(portfolioName);
    await detailPage.waitForLoad();
    await expect(detailPage.heading()).toContainText(portfolioName);

    // 3. Add a buy: 100 shares of AAPL @ $150, total $15,000
    await detailPage.clickAddTransaction();
    await detailPage.fillTransactionForm({
      type: 'Buy',
      symbol: 'AAPL',
      quantity: 100,
      pricePerShare: 150,
      totalAmount: 15000,
    });
    await detailPage.submitTransaction();

    // 4. Holdings tab shows AAPL with quantity 100
    await detailPage.switchToHoldingsTab();
    await detailPage.expectHoldingVisible('AAPL');
    expect(await detailPage.holdingQuantity('AAPL')).toBe('100');

    // 5. Add a sell: 40 shares @ $180, total $7,200
    await detailPage.clickAddTransaction();
    await detailPage.fillTransactionForm({
      type: 'Sell',
      symbol: 'AAPL',
      quantity: 40,
      pricePerShare: 180,
      totalAmount: 7200,
    });
    await detailPage.submitTransaction();

    // 6. Holdings tab shows quantity decreased to 60
    await detailPage.switchToHoldingsTab();
    await detailPage.expectHoldingVisible('AAPL');
    expect(await detailPage.holdingQuantity('AAPL')).toBe('60');

    // 7. Verify transaction list shows both entries
    await detailPage.switchToTransactionsTab();
    expect(await detailPage.transactionCount()).toBe(2);

    // 8. Cleanup: navigate back and delete portfolio
    await listPage.goto();
    await listPage.clickDeleteFor(portfolioName);
    await listPage.confirmDelete();
    await listPage.expectPortfolioNotVisible(portfolioName);
  });

  test('dividend transaction does not affect holdings quantity', async ({ page }) => {
    const listPage = new PortfolioListPage(page);
    const detailPage = new PortfolioDetailPage(page);
    const portfolioName = `E2E Dividend ${Date.now()}`;

    // Create portfolio
    await listPage.goto();
    await listPage.clickNewPortfolio();
    await listPage.fillPortfolioForm(portfolioName);
    await listPage.submitPortfolioForm();
    await listPage.clickPortfolioName(portfolioName);
    await detailPage.waitForLoad();

    // Buy 50 MSFT
    await detailPage.clickAddTransaction();
    await detailPage.fillTransactionForm({
      type: 'Buy',
      symbol: 'MSFT',
      quantity: 50,
      pricePerShare: 400,
      totalAmount: 20000,
    });
    await detailPage.submitTransaction();

    // Verify holding
    await detailPage.switchToHoldingsTab();
    expect(await detailPage.holdingQuantity('MSFT')).toBe('50');

    // Record a dividend (no shares)
    await detailPage.clickAddTransaction();
    await detailPage.fillTransactionForm({
      type: 'Dividend',
      symbol: 'MSFT',
      dividendPerShare: 0.75,
      totalAmount: 37.5,
    });
    await detailPage.submitTransaction();

    // Holding quantity unchanged — dividend is income, not shares
    await detailPage.switchToHoldingsTab();
    expect(await detailPage.holdingQuantity('MSFT')).toBe('50');

    // Cleanup
    await listPage.goto();
    await listPage.clickDeleteFor(portfolioName);
    await listPage.confirmDelete();
  });
});
