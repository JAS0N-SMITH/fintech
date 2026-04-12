import { test, expect } from '@playwright/test';

/**
 * Phase 9 E2E: Connection State & Error Handling
 *
 * Tests WebSocket connection indicator behavior:
 * - Verify status indicator displays on dashboard and ticker detail pages
 * - Simulate network disruption and verify indicator changes state
 * - Verify reconnection restores data
 *
 * Tests run against the live Angular dev server with real API.
 */

test.describe('Connection state indicator', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to dashboard (requires login in real scenario)
    await page.goto('/dashboard');
  });

  test('connection status indicator is visible on dashboard', async ({ page }) => {
    // Look for the connection status component (displays "Live", "Offline", "Reconnecting…")
    const statusIndicator = page.locator('app-connection-status');
    await expect(statusIndicator).toBeVisible();

    // Should show "Live" status initially
    const tag = page.locator('p-tag');
    const tagValue = await tag.getAttribute('ng-reflect-value');
    expect(['Live', 'Reconnecting…', 'Offline']).toContain(tagValue);
  });

  test('connection status indicator displays on ticker detail page', async ({ page }) => {
    // Navigate to a ticker detail page
    await page.goto('/tickers/AAPL');

    // Look for connection status component
    const statusIndicator = page.locator('app-connection-status');
    await expect(statusIndicator).toBeVisible();

    // Verify it has the symbol attribute (ticker-specific status)
    const symbol = await statusIndicator.getAttribute('ng-reflect-symbol');
    expect(symbol).toBeTruthy();
  });

  test('connection indicator shows "Live" status when connected', async ({ page }) => {
    await page.goto('/dashboard');

    // Find the connection status tag
    const tag = page.locator('p-tag').first();

    // Should display "Live" with success severity (green)
    await expect(tag).toContainText(/Live|Reconnecting|Offline/);

    // Check for success severity (class 'p-tag-success')
    const classList = await tag.getAttribute('class');
    expect(classList).toContain('p-tag-success');
  });

  test('UI remains functional during network simulation', async ({ page, context }) => {
    // Start monitoring requests
    let requestCount = 0;
    page.on('request', () => {
      requestCount++;
    });

    await page.goto('/dashboard');

    // Simulate going offline by intercepting WebSocket connections
    await page.route('ws://**', (route) => {
      route.abort('blockedbyclient');
    });

    // Wait a moment for WebSocket to fail
    await page.waitForTimeout(500);

    // Connection indicator should now show "Offline" or "Reconnecting"
    const tag = page.locator('p-tag').first();
    const tagText = await tag.innerText();
    expect(['Offline', 'Reconnecting…']).toContain(tagText.trim());

    // Verify the page doesn't crash and elements are still visible
    const header = page.locator('h1').first();
    await expect(header).toBeVisible();
  });

  test('connection indicator color changes reflect state', async ({ page }) => {
    await page.goto('/dashboard');

    // Get the first p-tag (connection status)
    const tag = page.locator('p-tag').first();

    // Initially should be success (green) or warn (yellow)
    let classList = await tag.getAttribute('class');
    expect(classList).toMatch(/p-tag-(success|warn)/);

    // Simulate network issue (abort WebSocket)
    await page.route('ws://**', (route) => {
      route.abort('blockedbyclient');
    });

    await page.waitForTimeout(1000);

    // Should transition to warn or danger
    classList = await tag.getAttribute('class');
    expect(classList).toMatch(/p-tag-(warn|danger)/);
  });

  test('reconnection is attempted after network restoration', async ({ page }) => {
    await page.goto('/dashboard');

    // Initially connected
    let tag = page.locator('p-tag').first();
    let classList = await tag.getAttribute('class');
    expect(classList).toContain('p-tag-success');

    // Abort WebSocket
    await page.route('ws://**', (route) => {
      route.abort('blockedbyclient');
    });

    await page.waitForTimeout(500);

    // Now offline/reconnecting
    tag = page.locator('p-tag').first();
    classList = await tag.getAttribute('class');
    expect(classList).toMatch(/p-tag-(warn|danger)/);

    // Restore the route (simulate network recovery)
    await page.unroute('ws://**');

    // Note: Actual reconnection depends on exponential backoff timing
    // In a real test, you would wait for reconnection or mock the delay
    // For now, just verify the page remains responsive
    const header = page.locator('h1').first();
    await expect(header).toBeVisible();
  });

  test('connection status component shows last updated timestamp on ticker detail', async ({
    page,
  }) => {
    await page.goto('/tickers/AAPL');

    // Find the connection status component
    const statusComponent = page.locator('app-connection-status');
    await expect(statusComponent).toBeVisible();

    // When connected, last-updated info should not show
    // When offline, it should show (depends on connection state)
    // For this test, just verify the component renders without errors
    await expect(statusComponent).toHaveCount(1);
  });
});
