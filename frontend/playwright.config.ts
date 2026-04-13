import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E configuration.
 *
 * Tests run against the Angular dev server (ng serve).
 * They require a running Go API and a seeded Supabase test project.
 *
 * Environment variables consumed by tests:
 *   E2E_USER_EMAIL    — test user email (default: test@example.com)
 *   E2E_USER_PASSWORD — test user password (default: testpassword123)
 *   BASE_URL          — override the app URL (default: http://localhost:4200)
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: false, // serial for now — shared backend state
  forbidOnly: !!process.env['CI'],
  retries: process.env['CI'] ? 1 : 0,
  workers: 1,
  reporter: [['html', { open: 'never' }], ['list']],

  use: {
    baseURL: process.env['BASE_URL'] ?? 'http://localhost:4200',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  // Snapshot directories for visual regression
  snapshotDir: './e2e/snapshots',
  snapshotPathTemplate: '{snapshotDir}/{testFileDir}/{testFileName}-{platform}{ext}',

  projects: [
    // Setup project: log in once as regular user and save auth state
    {
      name: 'setup',
      testMatch: '**/auth.setup.ts',
    },

    // Setup project: log in once as admin and save auth state
    {
      name: 'admin-setup',
      testMatch: '**/admin.setup.ts',
    },

    // Main test suite depends on regular user auth setup
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'e2e/.auth/user.json',
      },
      dependencies: ['setup'],
    },

    // Admin test suite depends on admin auth setup
    {
      name: 'admin',
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'e2e/.auth/admin.json',
      },
      testMatch: '**/admin.spec.ts',
      dependencies: ['admin-setup'],
    },
  ],

  // Start the Angular dev server automatically when running locally
  webServer: {
    command: 'ng serve',
    url: 'http://localhost:4200',
    reuseExistingServer: true,
    timeout: 120_000,
    stdout: 'ignore',
    stderr: 'pipe',
  },
});
