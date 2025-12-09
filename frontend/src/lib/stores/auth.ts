import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { api } from '$lib/api/client';

/**
 * User information from API
 */
export interface User {
	id: string;
	email: string;
	name: string;
	role: 'owner' | 'admin' | 'member';
	tenantId: string;
	tenantName: string;
	avatarUrl?: string;
}

/**
 * Login response from backend
 * NOTE: refresh_token is now sent as httpOnly cookie, not in response body
 */
interface LoginResponse {
	user: {
		id: string;
		email: string;
		name: string;
		role: string;
	};
	tenant_id: string;
	access_token: string;
	token_type: string;
	expires_in: number;
}

/**
 * Register response from backend
 * NOTE: refresh_token is now sent as httpOnly cookie, not in response body
 */
interface RegisterResponse {
	tenant: {
		id: string;
		name: string;
		slug: string;
	};
	user: {
		id: string;
		email: string;
		name: string;
		role: string;
	};
	access_token: string;
	token_type: string;
	expires_in: number;
}

/**
 * Refresh response from backend
 */
interface RefreshResponse {
	access_token: string;
	token_type: string;
	expires_in: number;
}

/**
 * Authentication state
 *
 * SECURITY: Access tokens are stored in memory only (not localStorage).
 * Refresh tokens are stored as httpOnly cookies by the backend and are
 * not accessible to JavaScript. This protects against XSS attacks.
 */
interface AuthState {
	user: User | null;
	isLoading: boolean;
	isInitialized: boolean;
	error: string | null;
}

const initialState: AuthState = {
	user: null,
	isLoading: false,
	isInitialized: false,
	error: null
};

/**
 * Create the auth store
 *
 * SECURITY NOTES:
 * - Access tokens stored in memory only (cleared on page refresh)
 * - Refresh tokens stored as httpOnly cookies (not accessible to JS)
 * - On page load, we call /refresh to get a new access token from the cookie
 * - This approach protects against XSS token theft
 */
function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>(initialState);

	// Store access token in memory only (not localStorage)
	// This is intentional for security - XSS cannot steal memory-only tokens

	return {
		subscribe,

		/**
		 * Initialize auth state by refreshing session from httpOnly cookie
		 * The refresh token is automatically sent as a cookie by the browser
		 */
		async initialize() {
			if (!browser) return;

			update((s) => ({ ...s, isLoading: true }));

			try {
				// Try to refresh the access token
				// The refresh token is sent automatically as httpOnly cookie
				const response = await api.post<RefreshResponse>('/api/v1/auth/refresh', {});

				// Store access token in memory (via api client)
				api.setAccessToken(response.access_token);

				// Get current user info
				const user = await api.get<User>('/api/v1/auth/me');
				update((s) => ({
					...s,
					user,
					isLoading: false,
					isInitialized: true,
					error: null
				}));
			} catch {
				// Not authenticated or token expired
				api.setAccessToken(null);
				update((s) => ({
					...s,
					user: null,
					isLoading: false,
					isInitialized: true,
					error: null
				}));
			}
		},

		/**
		 * Login with email and password
		 * Refresh token is set as httpOnly cookie by backend
		 */
		async login(email: string, password: string) {
			update((s) => ({ ...s, isLoading: true, error: null }));

			try {
				const response = await api.post<LoginResponse>('/api/v1/auth/login', {
					email,
					password
				});

				// Store access token in memory only
				api.setAccessToken(response.access_token);

				// Fetch full user info including tenant name
				const userInfo = await api.get<User>('/api/v1/auth/me');

				update((s) => ({
					...s,
					user: userInfo,
					isLoading: false,
					error: null
				}));

				return { success: true };
			} catch (error) {
				const message = api.isApiError(error) ? error.message : 'Login failed';
				update((s) => ({
					...s,
					isLoading: false,
					error: message
				}));
				return { success: false, error: message };
			}
		},

		/**
		 * Register a new account
		 * Refresh token is set as httpOnly cookie by backend
		 */
		async register(data: {
			email: string;
			password: string;
			name: string;
			tenantName: string;
			tenantSlug?: string;
		}) {
			update((s) => ({ ...s, isLoading: true, error: null }));

			try {
				const response = await api.post<RegisterResponse>('/api/v1/auth/register', {
					tenant_name: data.tenantName,
					tenant_slug: data.tenantSlug || data.tenantName.toLowerCase().replace(/\s+/g, '-'),
					name: data.name,
					email: data.email,
					password: data.password
				});

				// Store access token in memory only
				api.setAccessToken(response.access_token);

				// Build user from response
				const user: User = {
					id: response.user.id,
					email: response.user.email,
					name: response.user.name,
					role: response.user.role as 'owner' | 'admin' | 'member',
					tenantId: response.tenant.id,
					tenantName: response.tenant.name
				};

				update((s) => ({
					...s,
					user,
					isLoading: false,
					error: null
				}));

				return { success: true };
			} catch (error) {
				const message = api.isApiError(error) ? error.message : 'Registration failed';
				update((s) => ({
					...s,
					isLoading: false,
					error: message
				}));
				return { success: false, error: message };
			}
		},

		/**
		 * Logout and clear session
		 * Backend will clear the httpOnly refresh token cookie
		 */
		async logout() {
			try {
				// Backend reads refresh token from cookie and clears it
				await api.post('/api/v1/auth/logout', {});
			} catch {
				// Ignore errors - we're logging out anyway
			}

			// Clear access token from memory
			api.setAccessToken(null);
			set(initialState);

			if (browser) {
				goto('/login');
			}
		},

		/**
		 * Request password reset
		 */
		async requestPasswordReset(email: string) {
			try {
				await api.post('/api/v1/auth/forgot-password', { email });
				return { success: true };
			} catch (error) {
				const message = api.isApiError(error) ? error.message : 'Request failed';
				return { success: false, error: message };
			}
		},

		/**
		 * Reset password with token
		 */
		async resetPassword(token: string, password: string) {
			try {
				await api.post('/api/v1/auth/reset-password', { token, password });
				return { success: true };
			} catch (error) {
				const message = api.isApiError(error) ? error.message : 'Reset failed';
				return { success: false, error: message };
			}
		},

		/**
		 * Update user profile
		 */
		async updateProfile(data: { name?: string; avatarUrl?: string }) {
			const state = get({ subscribe });
			if (!state.user) return { success: false, error: 'Not authenticated' };

			try {
				const user = await api.patch<User>('/api/v1/auth/profile', data);
				update((s) => ({ ...s, user }));
				return { success: true };
			} catch (error) {
				const message = api.isApiError(error) ? error.message : 'Update failed';
				return { success: false, error: message };
			}
		},

		/**
		 * Change password
		 */
		async changePassword(currentPassword: string, newPassword: string) {
			try {
				await api.post('/api/v1/auth/change-password', {
					currentPassword,
					newPassword
				});
				return { success: true };
			} catch (error) {
				const message = api.isApiError(error) ? error.message : 'Password change failed';
				return { success: false, error: message };
			}
		},

		/**
		 * Get current access token (for WebSocket auth)
		 * Returns the in-memory access token
		 */
		getAccessToken(): string | null {
			return api.getAccessToken();
		},

		/**
		 * Clear error state
		 */
		clearError() {
			update((s) => ({ ...s, error: null }));
		}
	};
}

export const auth = createAuthStore();

// Derived stores for convenience
export const user = derived(auth, ($auth) => $auth.user);
export const isAuthenticated = derived(auth, ($auth) => !!$auth.user);
export const isLoading = derived(auth, ($auth) => $auth.isLoading);
export const isInitialized = derived(auth, ($auth) => $auth.isInitialized);
export const authError = derived(auth, ($auth) => $auth.error);

// Role-based access helpers
export const isOwner = derived(auth, ($auth) => $auth.user?.role === 'owner');
export const isAdmin = derived(auth, ($auth) =>
	$auth.user?.role === 'owner' || $auth.user?.role === 'admin'
);
