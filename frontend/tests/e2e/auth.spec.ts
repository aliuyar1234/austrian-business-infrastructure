import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
	test.describe('Login Page', () => {
		test('should display login form', async ({ page }) => {
			await page.goto('/login');

			// Check page title
			await expect(page).toHaveTitle(/Login/);

			// Check form elements exist
			await expect(page.locator('input[type="email"]')).toBeVisible();
			await expect(page.locator('input[type="password"]')).toBeVisible();
			await expect(page.locator('button[type="submit"]')).toBeVisible();
			await expect(page.locator('button[type="submit"]')).toContainText('Sign in');
		});

		test('should show validation errors for empty form', async ({ page }) => {
			await page.goto('/login');

			// Try to submit empty form
			await page.click('button[type="submit"]');

			// Browser should show validation (required fields)
			const emailInput = page.locator('input[type="email"]');
			const isInvalid = await emailInput.evaluate((el: HTMLInputElement) => !el.validity.valid);
			expect(isInvalid).toBe(true);
		});

		test('should show error for invalid credentials', async ({ page }) => {
			await page.goto('/login');

			await page.fill('input[type="email"]', 'invalid@test.com');
			await page.fill('input[type="password"]', 'wrongpassword');
			await page.click('button[type="submit"]');

			// Wait for error message
			await expect(page.locator('[class*="error"]')).toBeVisible({ timeout: 5000 });
		});

		test('should have link to forgot password', async ({ page }) => {
			await page.goto('/login');

			const forgotLink = page.locator('a[href="/forgot-password"]');
			await expect(forgotLink).toBeVisible();
		});

		test('should have link to register', async ({ page }) => {
			await page.goto('/login');

			const registerLink = page.locator('a[href="/register"]');
			await expect(registerLink).toBeVisible();
		});

		test('should have OAuth buttons', async ({ page }) => {
			await page.goto('/login');

			// Check for Google and Microsoft OAuth buttons
			await expect(page.locator('button:has-text("Google")')).toBeVisible();
			await expect(page.locator('button:has-text("Microsoft")')).toBeVisible();
		});
	});

	test.describe('Registration Page', () => {
		test('should display registration form', async ({ page }) => {
			await page.goto('/register');

			await expect(page).toHaveTitle(/Register/);

			await expect(page.locator('input#name')).toBeVisible();
			await expect(page.locator('input#email')).toBeVisible();
			await expect(page.locator('input#password')).toBeVisible();
			await expect(page.locator('input#confirmPassword')).toBeVisible();
		});

		test('should show password strength indicator', async ({ page }) => {
			await page.goto('/register');

			// Type a weak password
			await page.fill('input#password', 'weak');
			await expect(page.locator('text=Weak')).toBeVisible();

			// Type a stronger password
			await page.fill('input#password', 'StrongPass123!');
			await expect(page.locator('text=Strong')).toBeVisible();
		});

		test('should validate password match', async ({ page }) => {
			await page.goto('/register');

			await page.fill('input#name', 'Test User');
			await page.fill('input#email', 'test@example.com');
			await page.fill('input#password', 'Password123!');
			await page.fill('input#confirmPassword', 'DifferentPassword123!');

			await page.click('button[type="submit"]');

			// Should show password mismatch error
			await expect(page.locator('text=Passwords do not match')).toBeVisible({ timeout: 5000 });
		});
	});

	test.describe('Forgot Password Page', () => {
		test('should display forgot password form', async ({ page }) => {
			await page.goto('/forgot-password');

			await expect(page).toHaveTitle(/Forgot Password/);
			await expect(page.locator('input[type="email"]')).toBeVisible();
			await expect(page.locator('button[type="submit"]')).toContainText('Send reset link');
		});

		test('should show success message after submission', async ({ page }) => {
			await page.goto('/forgot-password');

			await page.fill('input[type="email"]', 'user@example.com');
			await page.click('button[type="submit"]');

			// Wait for success state
			await expect(page.locator('text=Check your email')).toBeVisible({ timeout: 5000 });
		});
	});

	test.describe('Protected Routes', () => {
		test('should redirect to login when not authenticated', async ({ page }) => {
			await page.goto('/');

			// Should redirect to login
			await expect(page).toHaveURL(/\/login/);
		});

		test('should redirect to login from protected pages', async ({ page }) => {
			const protectedRoutes = ['/accounts', '/documents', '/uva', '/invoices', '/settings'];

			for (const route of protectedRoutes) {
				await page.goto(route);
				await expect(page).toHaveURL(/\/login/);
			}
		});
	});
});
