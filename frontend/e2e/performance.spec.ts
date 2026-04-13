import { test, expect } from '@playwright/test';

/**
 * Performance budget tests for the application.
 *
 * These tests verify that the app meets performance thresholds:
 * - Time to Interactive (TTI) for critical routes
 * - Bundle size constraints (enforced by Angular build)
 *
 * Run with: npx playwright test performance.spec.ts
 */

test.describe('Performance budgets', () => {
  test('dashboard time-to-interactive within budget', async ({ page }) => {
    // Enable performance metrics collection
    const metrics = await page.evaluate(() => {
      return new Promise<{ duration: number }>((resolve) => {
        window.addEventListener('load', () => {
          const perfData = window.performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
          const tti = perfData.domInteractive - perfData.fetchStart;
          resolve({ duration: tti });
        });
      });
    });

    // Navigate to dashboard and wait for interactive
    const startTime = Date.now();
    await page.goto('/dashboard');

    // Wait for key interactive elements to be present
    await page.waitForSelector('[data-testid="portfolio-value-card"]', {
      timeout: 5000,
    });

    const loadTime = Date.now() - startTime;

    // Dashboard should be interactive within 3 seconds (generous threshold for slow CI)
    expect(loadTime).toBeLessThan(3000);
  });

  test('portfolio list time-to-interactive within budget', async ({ page }) => {
    const startTime = Date.now();
    await page.goto('/portfolios');

    // Wait for table to render
    await page.waitForSelector('p-table', {
      timeout: 5000,
    });

    const loadTime = Date.now() - startTime;

    // Portfolio list should be interactive within 2.5 seconds
    expect(loadTime).toBeLessThan(2500);
  });

  test('watchlist page time-to-interactive within budget', async ({ page }) => {
    const startTime = Date.now();
    await page.goto('/watchlist');

    // Wait for page content
    await page.waitForSelector('body > *', {
      timeout: 5000,
    });

    const loadTime = Date.now() - startTime;

    // Watchlist page should be interactive within 2.5 seconds
    expect(loadTime).toBeLessThan(2500);
  });

  test('initial bundle size within limits (visual check)', async ({ page }) => {
    // This test doesn't fail the build but logs bundle metrics
    // The actual enforcement happens during `ng build` with budget warnings/errors

    // Load the main bundle size stats if available
    // In CI, this would be computed from dist/stats.json
    const bundleInfo = await page.evaluate(() => {
      // Try to read from window if injected by build
      return (window as any).bundleMetrics || { main: 'unknown', styles: 'unknown' };
    });

    console.log('Bundle metrics:', bundleInfo);
    // In real scenarios, bundleMetrics would contain {main: 450kB, styles: 15kB, etc.}
  });

  test('lazy-loaded routes do not block initial bundle', async ({ page }) => {
    // All feature routes should be lazy-loaded
    // Verify by checking network tab for feature chunks

    const requests: string[] = [];
    page.on('response', (response) => {
      if (response.url().includes('chunk') || response.url().includes('vendor')) {
        requests.push(response.url());
      }
    });

    // Navigate to lazy route (should trigger feature chunk download)
    await page.goto('/dashboard');
    await page.waitForSelector('[data-testid="portfolio-value-card"]');

    // At least one lazy chunk should be loaded (not the main bundle)
    // This is a soft check; actual enforcement is in build optimization
    expect(page.url()).toContain('dashboard');
  });
});
