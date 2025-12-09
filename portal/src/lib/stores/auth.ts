import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { api } from '$lib/api/client';

/**
 * Client info from API
 */
interface ClientInfo {
	id: string;
	name: string;
	email: string;
	tenantId: string;
	tenantName: string;
}

/**
 * Auth state
 *
 * SECURITY: Access tokens are stored in memory only (not localStorage).
 * Refresh tokens are stored as httpOnly cookies by the backend and are
 * not accessible to JavaScript. This protects against XSS attacks.
 */
interface AuthState {
	isAuthenticated: boolean;
	client: ClientInfo | null;
	loading: boolean;
	initialized: boolean;
	error: string | null;
}

const initialState: AuthState = {
	isAuthenticated: false,
	client: null,
	loading: false,
	initialized: false,
	error: null
};

/**
 * Create the portal auth store
 *
 * SECURITY NOTES:
 * - Access tokens stored in memory only (cleared on page refresh)
 * - Refresh tokens stored as httpOnly cookies (not accessible to JS)
 * - On page load, we call /refresh to get a new access token from the cookie
 * - This approach protects against XSS token theft
 */
function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>(initialState);

	return {
		subscribe,

		/**
		 * Initialize auth state by refreshing session from httpOnly cookie
		 * The refresh token is automatically sent as a cookie by the browser
		 */
		async initialize() {
			if (!browser) return;

			update((s) => ({ ...s, loading: true }));

			try {
				// Try to refresh the access token
				// The refresh token is sent automatically as httpOnly cookie
				await api.refreshToken();

				// Get current client info
				const me = await api.getMe();
				update((s) => ({
					...s,
					isAuthenticated: true,
					client: {
						id: me.id,
						name: me.name,
						email: me.email,
						tenantId: me.tenant_id,
						tenantName: me.tenant_name
					},
					loading: false,
					initialized: true,
					error: null
				}));
			} catch {
				// Not authenticated or token expired
				api.setAccessToken(null);
				update((s) => ({
					...s,
					isAuthenticated: false,
					client: null,
					loading: false,
					initialized: true,
					error: null
				}));
			}
		},

		/**
		 * Login with email and password
		 * Refresh token is set as httpOnly cookie by backend
		 */
		async login(email: string, password: string) {
			update((s) => ({ ...s, loading: true, error: null }));

			try {
				// Login - backend sets httpOnly cookie, returns access token
				await api.login(email, password);

				// Fetch client profile
				const me = await api.getMe();
				update((s) => ({
					...s,
					isAuthenticated: true,
					client: {
						id: me.id,
						name: me.name,
						email: me.email,
						tenantId: me.tenant_id,
						tenantName: me.tenant_name
					},
					loading: false,
					error: null
				}));

				return { success: true };
			} catch (e: any) {
				const message = e.message || 'Login failed';
				update((s) => ({
					...s,
					loading: false,
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
				await api.logout();
			} catch {
				// Ignore errors - we're logging out anyway
			}

			// Clear access token from memory
			api.setAccessToken(null);
			set({ ...initialState, initialized: true });

			if (browser) {
				goto('/login');
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
export const isAuthenticated = derived(auth, ($auth) => $auth.isAuthenticated);
export const client = derived(auth, ($auth) => $auth.client);
export const isLoading = derived(auth, ($auth) => $auth.loading);
export const isInitialized = derived(auth, ($auth) => $auth.initialized);
export const authError = derived(auth, ($auth) => $auth.error);
