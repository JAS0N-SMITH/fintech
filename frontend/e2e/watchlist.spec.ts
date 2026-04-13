import { test, expect } from '@playwright/test';
import { injectAxe, checkA11y } from 'axe-playwright';
import { WatchlistPage } from './pages/watchlist.page';

/**
 * E2E tests for the watchlist feature.
 *
 * Tests the full user flow:
 * 1. Create a watchlist
 * 2. Add a ticker to the watchlist
 * 3. Verify live price updates
 * 4. Remove the ticker
 * 5. Delete the watchlist
 */
test.describe('Watchlist Feature', () => {
  let watchlistPage: WatchlistPage;

  test.beforeEach(async ({ page }) => {
    // Login before each test (adjust based on your auth setup)
    // For now, assume user is already authenticated via session
    watchlistPage = new WatchlistPage(page);
    await watchlistPage.goto();
  });

  test('should create a watchlist', async () => {
    const watchlistName = `Test Watchlist ${Date.now()}`;

    // Count initial watchlists
    const initialCount = await watchlistPage.getWatchlistCount();

    // Create new watchlist
    await watchlistPage.createWatchlist(watchlistName);

    // Verify watchlist was added
    const newCount = await watchlistPage.getWatchlistCount();
    expect(newCount).toBe(initialCount + 1);

    // Verify success message appears
    await expect(watchlistPage.page.getByText(/created/i)).toBeVisible();
  });

  test('should add a ticker to a watchlist', async ({ page }) => {
    const watchlistName = `Test Watchlist ${Date.now()}`;

    // Create watchlist
    await watchlistPage.createWatchlist(watchlistName);

    // Open the watchlist
    await watchlistPage.openWatchlist(watchlistName);

    // Initially, no items
    let itemCount = await watchlistPage.getItemCount();
    expect(itemCount).toBe(0);

    // Add AAPL ticker
    await watchlistPage.addTicker('AAPL');

    // Verify item was added
    itemCount = await watchlistPage.getItemCount();
    expect(itemCount).toBe(1);

    // Verify success message
    await expect(page.getByText(/added/i)).toBeVisible();

    // Verify ticker is displayed
    const aaplRow = await watchlistPage.getItemBySymbol('AAPL');
    await expect(aaplRow).toBeVisible();
  });

  test('should show live prices for ticker', async ({ page }) => {
    const watchlistName = `Test Watchlist ${Date.now()}`;

    // Create watchlist
    await watchlistPage.createWatchlist(watchlistName);

    // Open the watchlist
    await watchlistPage.openWatchlist(watchlistName);

    // Add AAPL ticker
    await watchlistPage.addTicker('AAPL');

    // Wait for price to update from WebSocket
    await watchlistPage.waitForPriceUpdate('AAPL');

    // Get the price text
    const priceText = await watchlistPage.getItemPrice('AAPL');

    // Verify price is displayed (should contain $ and a number)
    expect(priceText).toMatch(/\$\d+\.\d{2}/);
  });

  test('should remove a ticker from watchlist', async ({ page }) => {
    const watchlistName = `Test Watchlist ${Date.now()}`;

    // Create watchlist
    await watchlistPage.createWatchlist(watchlistName);

    // Open the watchlist
    await watchlistPage.openWatchlist(watchlistName);

    // Add AAPL ticker
    await watchlistPage.addTicker('AAPL');

    // Verify item was added
    let itemCount = await watchlistPage.getItemCount();
    expect(itemCount).toBe(1);

    // Remove the ticker
    await watchlistPage.removeTicker('AAPL');

    // Verify item was removed
    itemCount = await watchlistPage.getItemCount();
    expect(itemCount).toBe(0);

    // Verify success message
    await expect(page.getByText(/removed/i)).toBeVisible();
  });

  test('should delete a watchlist', async ({ page }) => {
    const watchlistName = `Test Watchlist ${Date.now()}`;

    // Create watchlist
    await watchlistPage.createWatchlist(watchlistName);

    // Verify watchlist exists
    await expect(
      page.getByRole('button', { name: `View ${watchlistName}` })
    ).toBeVisible();

    // Delete the watchlist
    await watchlistPage.deleteWatchlist(watchlistName);

    // Verify watchlist is gone
    await expect(
      page.getByRole('button', { name: `View ${watchlistName}` })
    ).not.toBeVisible();

    // Verify success message
    await expect(page.getByText(/deleted/i)).toBeVisible();
  });

  test('should rename a watchlist', async ({ page }) => {
    const watchlistName = `Test Watchlist ${Date.now()}`;
    const newName = `Renamed Watchlist ${Date.now()}`;

    // Create watchlist
    await watchlistPage.createWatchlist(watchlistName);

    // Rename the watchlist
    await watchlistPage.renameWatchlist(watchlistName, newName);

    // Verify new name is displayed
    await expect(
      page.getByRole('button', { name: `View ${newName}` })
    ).toBeVisible();

    // Verify old name is gone
    await expect(
      page.getByRole('button', { name: `View ${watchlistName}` })
    ).not.toBeVisible();

    // Verify success message
    await expect(page.getByText(/renamed/i)).toBeVisible();
  });

  test('should navigate back from watchlist detail to list', async () => {
    const watchlistName = `Test Watchlist ${Date.now()}`;

    // Create and open watchlist
    await watchlistPage.createWatchlist(watchlistName);
    await watchlistPage.openWatchlist(watchlistName);

    // Click back button
    await watchlistPage.goBack();

    // Verify we're back on the list page
    await expect(watchlistPage.newWatchlistButton).toBeVisible();
    await expect(
      watchlistPage.page.getByRole('button', { name: `View ${watchlistName}` })
    ).toBeVisible();
  });

  test('should handle concurrent ticker operations', async () => {
    const watchlistName = `Test Watchlist ${Date.now()}`;

    // Create and open watchlist
    await watchlistPage.createWatchlist(watchlistName);
    await watchlistPage.openWatchlist(watchlistName);

    // Add multiple tickers
    await watchlistPage.addTicker('AAPL');
    await watchlistPage.addTicker('GOOGL');

    // Verify both are added
    let itemCount = await watchlistPage.getItemCount();
    expect(itemCount).toBe(2);

    // Remove one ticker
    await watchlistPage.removeTicker('AAPL');

    // Verify correct item was removed
    itemCount = await watchlistPage.getItemCount();
    expect(itemCount).toBe(1);

    // Verify GOOGL is still there
    const googlRow = await watchlistPage.getItemBySymbol('GOOGL');
    await expect(googlRow).toBeVisible();
  });
});

test.describe('Watchlist accessibility', () => {
  test('watchlist list is accessible', async ({ page }) => {
    const watchlistPage = new WatchlistPage(page);
    await watchlistPage.goto();

    // Inject axe and run accessibility checks
    await injectAxe(page);
    await checkA11y(page, null, {
      detailedReport: true,
    });
  });

  test('watchlist list visual regression', async ({ page }) => {
    const watchlistPage = new WatchlistPage(page);
    await watchlistPage.goto();

    // Capture page for visual regression
    await expect(page.locator('body')).toHaveScreenshot('watchlist-list-layout.png');
  });
});
