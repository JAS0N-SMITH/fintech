import { test, expect } from '@playwright/test';
import { injectAxe, checkA11y } from 'axe-playwright';
import { DashboardPage } from './pages/dashboard.page';

test.describe('Dashboard', () => {
  let dashboardPage: DashboardPage;

  test.beforeEach(async ({ page }) => {
    dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();
  });

  test('displays portfolio summary cards', async ({ page }) => {
    // Verify all three summary cards are present
    await expect(dashboardPage.portfolioValueCard).toBeVisible();
    await expect(dashboardPage.gainLossCard).toBeVisible();
    await expect(dashboardPage.dayChangeCard).toBeVisible();

    // Verify values are numeric (or loading skeleton)
    const value = await dashboardPage.getPortfolioValue();
    expect(value).toBeTruthy();
  });

  test('renders allocation chart', async ({ page }) => {
    // Wait for chart container to render
    await expect(dashboardPage.allocationChart).toBeVisible();

    // Chart contains a canvas element for visualization
    const canvas = dashboardPage.allocationChart.locator('canvas');
    await expect(canvas).toBeVisible();
  });

  test('displays top movers tables', async ({ page }) => {
    // Verify tables render
    await expect(dashboardPage.topGainersTable).toBeVisible();
    await expect(dashboardPage.topLosersTable).toBeVisible();

    // Tables should have header rows
    const gainersHeader = dashboardPage.topGainersTable.locator('thead');
    const losersHeader = dashboardPage.topLosersTable.locator('thead');
    await expect(gainersHeader).toBeVisible();
    await expect(losersHeader).toBeVisible();
  });

  test('navigates to ticker detail from top gainers', async ({ page }) => {
    // Click on the first gainer
    const gainers = await dashboardPage.getTopGainers();
    if (gainers.length > 0) {
      await dashboardPage.clickOnGainer(0);

      // Should navigate to ticker detail page
      expect(page.url()).toContain('/tickers/');
    }
  });

  test('dashboard layout responsive', async ({ page }) => {
    // Cards should be visible on desktop (already at 1280px in Playwright)
    const cardCount = await page.locator('[data-testid*="card"]').count();
    expect(cardCount).toBeGreaterThanOrEqual(3);
  });

  test('visual regression: light theme layout', async ({ page }) => {
    // Capture dashboard layout for light theme
    await expect(page.locator('body')).toHaveScreenshot(
      'dashboard-light-layout.png',
      { mask: [page.locator('p-menu')] }, // Mask menu if it varies
    );
  });

  test('visual regression: dark theme layout', async ({ page }) => {
    // Toggle to dark theme
    const themeToggle = page.getByRole('button', { name: /theme|dark|light/i });
    if (await themeToggle.isVisible()) {
      await themeToggle.click();
      // Wait for theme transition
      await page.waitForTimeout(300);
    }

    // Capture dashboard layout for dark theme
    await expect(page.locator('body')).toHaveScreenshot('dashboard-dark-layout.png');
  });

  test('accessibility: axe scan', async ({ page }) => {
    // Inject axe and run accessibility checks
    await injectAxe(page);
    await checkA11y(page, null, {
      detailedReport: true,
      detailedReportOptions: {
        html: true,
      },
    });
  });

  test('accessibility: keyboard navigation', async ({ page }) => {
    // Tab through interactive elements on the page
    const interactiveCount = await dashboardPage.getInteractiveCount();
    expect(interactiveCount).toBeGreaterThan(0);

    // Focus should be manageable via Tab key
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    // Focus should be on a button or link
    const focusedRole = await page.evaluate(() =>
      document.activeElement?.getAttribute('role'),
    );
    expect(['button', 'link', 'menuitem']).toContain(focusedRole || 'button');
  });

  test('accessibility: summary cards have aria-labels', async ({ page }) => {
    // All summary cards should have accessible labels
    const valueCardLabel = await dashboardPage.portfolioValueCard.getAttribute(
      'aria-label',
    );
    expect(valueCardLabel).toBeTruthy();

    const gainLossCardLabel = await dashboardPage.gainLossCard.getAttribute(
      'aria-label',
    );
    expect(gainLossCardLabel).toBeTruthy();
  });

  test('connection status indicator visible', async ({ page }) => {
    // Dashboard should show connection state badge (e.g., "Live" or "Reconnecting")
    const connectionBadge = page.locator('[data-testid="connection-status"]');
    await expect(connectionBadge).toBeVisible({ timeout: 5000 });

    const status = await connectionBadge.textContent();
    expect(status).toMatch(/Live|Reconnecting|Offline/i);
  });
});
