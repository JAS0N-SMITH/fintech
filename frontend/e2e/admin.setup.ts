import { test as setup, expect } from '@playwright/test';

/**
 * Admin auth setup project — runs once before admin tests.
 *
 * Logs in with the E2E admin test user and persists the Supabase session to
 * e2e/.auth/admin.json. Admin-specific E2E tests load this storageState so they
 * start already authenticated as an admin.
 *
 * Required environment variables (or defaults used in dev):
 *   E2E_ADMIN_EMAIL    — admin test account email
 *   E2E_ADMIN_PASSWORD — admin test account password
 */

const ADMIN_EMAIL = process.env['E2E_ADMIN_EMAIL'] ?? 'admin@example.com';
const ADMIN_PASSWORD = process.env['E2E_ADMIN_PASSWORD'] ?? 'adminpassword123';
const ADMIN_AUTH_FILE = 'e2e/.auth/admin.json';

setup('authenticate as admin', async ({ page }) => {
  await page.goto('/auth/login');

  await page.getByLabel('Email').fill(ADMIN_EMAIL);
  await page.getByLabel('Password').fill(ADMIN_PASSWORD);
  await page.getByRole('button', { name: /sign in/i }).click();

  // Wait until we land on the portfolio list — confirms auth succeeded
  await expect(page).toHaveURL(/\/portfolios/, { timeout: 15_000 });

  await page.context().storageState({ path: ADMIN_AUTH_FILE });
});
