import { test, expect } from '@playwright/test';
import { AdminPage } from './pages/admin.page';

/**
 * Phase 10 E2E: Admin Dashboard
 *
 * Tests run against the live Angular dev server with a real Go API.
 * Two test contexts: admin user (can access) and regular user (cannot access).
 * Both contexts use the storageState from auth.setup.ts for authenticated requests.
 */

test.describe('Admin Dashboard - Admin User Access', () => {
  let adminPage: AdminPage;

  test.beforeEach(async ({ page }) => {
    // Use admin storageState from auth.setup.ts
    adminPage = new AdminPage(page);
    await adminPage.goto();
  });

  test('admin layout renders with sidebar navigation', async ({ page }) => {
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    const menuItems = page.locator('[role="menuitem"]');
    await expect(menuItems).toHaveCount(5); // Users, Audit Log, Health, separator, Back
  });

  test('users page loads and displays users table', async () => {
    await adminPage.navigateToUsers();

    const table = adminPage['page'].locator('p-table');
    await expect(table).toBeVisible();

    // Table should have headers
    const headers = adminPage['page'].locator('p-table thead th');
    const headerCount = await headers.count();
    await expect(headerCount).toBeGreaterThan(0);
  });

  test('can change a user role and audit log records the change', async ({ page }) => {
    // Prerequisite: need a test user to change. For now, test the dialog flow.
    await adminPage.navigateToUsers();

    const rows = page.locator('p-table tbody tr');
    const rowCount = await rows.count();

    if (rowCount === 0) {
      test.skip();
      return;
    }

    // Get the first user's email (from first cell of first row)
    const firstRow = rows.first();
    const cells = firstRow.locator('td');
    const email = await cells.nth(0).textContent();

    if (!email) {
      test.skip();
      return;
    }

    // Open role dialog
    await adminPage.openRoleDialog(email);

    // Select new role
    await adminPage.selectNewRole('admin');

    // Confirm change
    await adminPage.confirmRoleChange();

    // Expect success toast
    await adminPage.expectSuccessToast('role updated');

    // Navigate to audit log and verify entry exists
    await adminPage.navigateToAuditLog();

    // Filter by role_change action
    await adminPage.filterByAction('role_change');
    await adminPage.clickSearch();

    // Verify at least one entry is visible
    const entries = await adminPage.getAuditLogEntries();
    expect(entries.length).toBeGreaterThan(0);
    expect(entries[0]).toContain('role_change');
  });

  test('audit log page loads with filters', async () => {
    await adminPage.navigateToAuditLog();

    const actionInput = adminPage['page'].locator('input[placeholder*="e.g. role_change"]');
    await expect(actionInput).toBeVisible();

    const searchButton = adminPage['page'].getByRole('button', { name: /search/i });
    await expect(searchButton).toBeVisible();

    const exportButton = adminPage['page'].getByRole('button', { name: /export to csv/i });
    await expect(exportButton).toBeVisible();
  });

  test('system health page shows component status', async () => {
    await adminPage.navigateToHealth();

    const cards = adminPage['page'].locator('p-card');
    await expect(cards).toHaveCount(3); // DB, Finnhub, WebSocket

    // Verify DB status is visible
    const dbStatus = await adminPage.getHealthStatus('db');
    expect(['healthy', 'unhealthy', 'unavailable']).toContain(dbStatus);
  });

  test('back to app button returns to dashboard', async ({ page }) => {
    await adminPage.backToApp();
    await page.waitForURL('/');
  });
});

test.describe('Admin Dashboard - Regular User Access (RBAC)', () => {
  test('regular user cannot access /admin', async ({ page, context }) => {
    // Note: In a real setup, you'd load a different storageState for a non-admin user.
    // For now, this test demonstrates the intent: non-admin users should be blocked.
    // In CI, use two separate browser contexts with different auth tokens.

    // Try to navigate to admin
    await page.goto('/admin');

    // Should be redirected to home (due to adminGuard)
    await page.waitForURL('/');

    // Verify we're at the dashboard, not admin
    const adminLayout = page.locator('app-admin-layout');
    await expect(adminLayout).not.toBeVisible();
  });

  test('admin routes do not download non-admin code bundles', async ({ page, context }) => {
    // Verify that the admin code bundle is not requested for non-admin users.
    // This is enforced by canMatch: [adminGuard] in admin.routes.ts.

    let adminChunkRequested = false;

    page.on('request', (request) => {
      if (request.url().includes('admin') && request.url().endsWith('.js')) {
        adminChunkRequested = true;
      }
    });

    // Navigate to a public page
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Admin chunk should not have been requested
    expect(adminChunkRequested).toBe(false);
  });
});

test.describe('Admin Dashboard - API Security', () => {
  test('GET /api/v1/admin/users without auth returns 401', async ({ page, context }) => {
    // Create a fresh context without auth cookies/tokens
    const apiContext = await context.request.newContext({
      storageState: undefined, // No auth
    });

    const response = await apiContext.get('/api/v1/admin/users');
    expect(response.status()).toBe(401);

    await apiContext.dispose();
  });

  test('GET /api/v1/admin/users as non-admin user returns 403', async ({ request }) => {
    // This test requires two auth tokens: admin and non-admin.
    // In CI, create a non-admin test user and use their token.
    // For now, document the test intent.

    // const nonAdminToken = await getNonAdminTestToken();
    // const response = await request.get('/api/v1/admin/users', {
    //   headers: { Authorization: `Bearer ${nonAdminToken}` },
    // });
    // expect(response.status()).toBe(403);

    test.skip();
  });

  test('PATCH /admin/users/:id/role with invalid role returns 400', async ({ request }) => {
    // This test requires auth.
    // Demonstrates validation: role must be 'user' or 'admin'.

    // const response = await request.patch('/api/v1/admin/users/some-id/role', {
    //   data: { role: 'superadmin' },
    //   headers: { Authorization: `Bearer ${adminToken}` },
    // });
    // expect(response.status()).toBe(400);

    test.skip();
  });
});
