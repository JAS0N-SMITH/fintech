import { test, expect } from '@playwright/test';

/**
 * Auth session E2E regression checks for cookie-backed restore.
 *
 * These tests run in the regular authenticated project (storageState from
 * auth.setup.ts). They verify a cold-start restore still succeeds when
 * local/session storage auth keys are removed.
 */
test.describe('Auth session restore', () => {
  test('restores session from cookie on reload when browser auth storage is cleared', async ({
    page,
  }) => {
    await page.goto('/portfolios');
    await expect(page).toHaveURL(/\/portfolios/);

    await page.evaluate(() => {
      const localKeys = Object.keys(localStorage).filter((k) => k.includes('-auth-token'));
      for (const key of localKeys) {
        localStorage.removeItem(key);
      }

      const sessionKeys = Object.keys(sessionStorage).filter((k) => k.includes('-auth-token'));
      for (const key of sessionKeys) {
        sessionStorage.removeItem(key);
      }
    });

    const sessionCall = page.waitForRequest(
      (req) => req.method() === 'GET' && req.url().includes('/api/v1/auth/session'),
    );

    await page.reload();
    await sessionCall;

    await expect(page).toHaveURL(/\/portfolios/);
    await expect(page.locator('p-table')).toBeVisible();
  });
});
