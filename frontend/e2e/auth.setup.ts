import { test as setup, expect } from '@playwright/test';

/**
 * Auth setup project — runs once before all other test projects.
 *
 * Logs in with the E2E test user and persists the Supabase session to
 * e2e/.auth/user.json. Subsequent tests load this storageState so they
 * start already authenticated.
 *
 * Required environment variables (or defaults used in dev):
 *   E2E_USER_EMAIL    — test account email
 *   E2E_USER_PASSWORD — test account password
 */

const EMAIL = process.env['E2E_USER_EMAIL'] ?? 'test@example.com';
const PASSWORD = process.env['E2E_USER_PASSWORD'] ?? 'testpassword123';
const AUTH_FILE = 'e2e/.auth/user.json';

setup('authenticate', async ({ page }) => {
  await page.goto('/auth/login');

  await page.getByLabel('Email').fill(EMAIL);
  await page.getByLabel('Password').fill(PASSWORD);
  await page.getByRole('button', { name: /sign in/i }).click();

  // Wait until we land on the portfolio list — confirms auth succeeded
  await expect(page).toHaveURL(/\/portfolios/, { timeout: 15_000 });

  await page.context().storageState({ path: AUTH_FILE });
});
