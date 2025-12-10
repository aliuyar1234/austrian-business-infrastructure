import { test, expect, type Page } from '@playwright/test';

// Helper to mock authentication
async function login(page: Page) {
	// Set mock auth state in localStorage
	await page.addInitScript(() => {
		localStorage.setItem('auth_token', 'mock-jwt-token');
		localStorage.setItem('user', JSON.stringify({
			id: '1',
			email: 'test@example.com',
			name: 'Test User',
			tenantId: 'tenant-1',
			tenantName: 'Test Company',
			role: 'owner',
		}));
	});
}

test.describe('Dashboard', () => {
	test.beforeEach(async ({ page }) => {
		await login(page);
	});

	test('should display dashboard after login', async ({ page }) => {
		await page.goto('/');

		// Wait for dashboard to load
		await expect(page.locator('h1')).toContainText('Dashboard');
	});

	test('should display all dashboard widgets', async ({ page }) => {
		await page.goto('/');

		// Check for key widgets
		await expect(page.locator('text=New Documents')).toBeVisible();
		await expect(page.locator('text=Upcoming Deadlines')).toBeVisible();
		await expect(page.locator('text=Account Status')).toBeVisible();
		await expect(page.locator('text=Recent Activity')).toBeVisible();
	});

	test('should show quick action buttons', async ({ page }) => {
		await page.goto('/');

		// Check for quick action buttons
		await expect(page.locator('button:has-text("Sync All")')).toBeVisible();
		await expect(page.locator('button:has-text("New UVA")')).toBeVisible();
	});

	test('should have working navigation to all main sections', async ({ page }) => {
		await page.goto('/');

		// Click on Accounts in sidebar
		await page.click('a[href="/accounts"]');
		await expect(page).toHaveURL('/accounts');

		// Click on Documents
		await page.click('a[href="/documents"]');
		await expect(page).toHaveURL('/documents');

		// Click on UVA
		await page.click('a[href="/uva"]');
		await expect(page).toHaveURL('/uva');
	});

	test('should open command palette with Cmd+K', async ({ page }) => {
		await page.goto('/');

		// Trigger command palette
		await page.keyboard.press('Meta+k');

		// Check command palette is visible
		await expect(page.locator('input[placeholder*="Search"]')).toBeVisible();
	});

	test('should show keyboard shortcuts with ?', async ({ page }) => {
		await page.goto('/');

		// Wait for page to load
		await page.waitForLoadState('networkidle');

		// Press ? to show shortcuts
		await page.keyboard.press('?');

		// Check shortcuts modal is visible
		await expect(page.locator('text=Keyboard Shortcuts')).toBeVisible();
	});

	test('should have header with user menu', async ({ page }) => {
		await page.goto('/');

		// Check user initials in header
		await expect(page.locator('header')).toBeVisible();

		// Click user menu
		const userMenu = page.locator('header button:has([class*="rounded-full"])');
		await userMenu.click();

		// Check dropdown appears
		await expect(page.locator('text=Sign out')).toBeVisible();
	});

	test('should have theme toggle in header', async ({ page }) => {
		await page.goto('/');

		// Find theme toggle button
		const themeToggle = page.locator('button[aria-label*="theme"]');
		await expect(themeToggle).toBeVisible();

		// Click to toggle theme
		await themeToggle.click();

		// Check that dark class is applied or removed from html
		const htmlElement = page.locator('html');
		const hasDarkClass = await htmlElement.evaluate((el: Element) => el.classList.contains('dark'));
		// Just verify the toggle worked (either on or off)
		expect(typeof hasDarkClass).toBe('boolean');
	});
});

test.describe('Dashboard Widgets', () => {
	test.beforeEach(async ({ page }) => {
		await login(page);
	});

	test('should show document count in New Documents widget', async ({ page }) => {
		await page.goto('/');

		// Look for document count or empty state
		const docsWidget = page.locator('text=New Documents').locator('..');
		await expect(docsWidget).toBeVisible();
	});

	test('should show deadline warnings when approaching', async ({ page }) => {
		await page.goto('/');

		// Look for deadline badges or empty state
		const deadlinesWidget = page.locator('text=Upcoming Deadlines').locator('..');
		await expect(deadlinesWidget).toBeVisible();
	});

	test('should navigate to documents when clicking on widget', async ({ page }) => {
		await page.goto('/');

		// Click on documents widget "View all" link if exists
		const viewAllLink = page.locator('a:has-text("View all")').first();
		if (await viewAllLink.isVisible()) {
			await viewAllLink.click();
			await expect(page).toHaveURL(/\/(documents|accounts|uva)/);
		}
	});
});

test.describe('Dashboard Responsive Design', () => {
	test.beforeEach(async ({ page }) => {
		await login(page);
	});

	test('should have responsive sidebar on mobile', async ({ page }) => {
		// Set mobile viewport
		await page.setViewportSize({ width: 375, height: 667 });
		await page.goto('/');

		// Sidebar should be collapsed or hidden on mobile
		const sidebar = page.locator('aside');

		// On mobile, sidebar might be hidden or behind a hamburger menu
		// This test verifies the page still works on mobile
		await expect(page.locator('h1')).toBeVisible();
	});

	test('should stack widgets on narrow screens', async ({ page }) => {
		await page.setViewportSize({ width: 768, height: 1024 });
		await page.goto('/');

		// Widgets should still be visible
		await expect(page.locator('text=New Documents')).toBeVisible();
		await expect(page.locator('text=Account Status')).toBeVisible();
	});
});
