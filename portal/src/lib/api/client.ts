const API_BASE = '/api/v1/portal';

interface ApiError {
	message: string;
	status: number;
}

/**
 * Portal API client with memory-only access token storage
 *
 * SECURITY: Access tokens are stored in memory only (not localStorage).
 * Refresh tokens are stored as httpOnly cookies by the backend and are
 * not accessible to JavaScript. This protects against XSS attacks.
 */
class PortalApiClient {
	// Store access token in memory only - XSS cannot steal memory-only tokens
	private accessToken: string | null = null;

	/**
	 * Set the access token (stored in memory only)
	 */
	setAccessToken(token: string | null) {
		this.accessToken = token;
	}

	/**
	 * Get the current access token (for WebSocket auth)
	 */
	getAccessToken(): string | null {
		return this.accessToken;
	}

	private async request<T>(
		endpoint: string,
		options: RequestInit = {}
	): Promise<T> {
		const url = `${API_BASE}${endpoint}`;

		const headers: Record<string, string> = {
			'Content-Type': 'application/json',
			...(options.headers as Record<string, string>)
		};

		// Add Authorization header if we have an access token
		if (this.accessToken) {
			headers['Authorization'] = `Bearer ${this.accessToken}`;
		}

		const response = await fetch(url, {
			...options,
			credentials: 'include', // Send httpOnly cookies
			headers
		});

		if (!response.ok) {
			if (response.status === 401) {
				// Try to refresh the token
				const refreshed = await this.tryRefresh();
				if (refreshed) {
					// Retry the request with new token
					headers['Authorization'] = `Bearer ${this.accessToken}`;
					const retryResponse = await fetch(url, {
						...options,
						credentials: 'include',
						headers
					});
					if (retryResponse.ok) {
						if (retryResponse.status === 204) {
							return undefined as T;
						}
						return retryResponse.json();
					}
				}
				// Refresh failed - redirect to login
				this.accessToken = null;
				window.location.href = '/login';
				throw new Error('Unauthorized');
			}

			const error: ApiError = {
				message: await response.text(),
				status: response.status
			};
			throw error;
		}

		if (response.status === 204) {
			return undefined as T;
		}

		return response.json();
	}

	/**
	 * Try to refresh the access token using httpOnly cookie
	 */
	private async tryRefresh(): Promise<boolean> {
		try {
			const response = await fetch(`${API_BASE}/auth/refresh`, {
				method: 'POST',
				credentials: 'include', // Send httpOnly refresh cookie
				headers: {
					'Content-Type': 'application/json'
				}
			});

			if (response.ok) {
				const data = await response.json();
				this.accessToken = data.access_token;
				return true;
			}
			return false;
		} catch {
			return false;
		}
	}

	// Auth
	async login(email: string, password: string) {
		// Login response sets refresh token as httpOnly cookie
		// and returns access token in body
		const response = await fetch(`${API_BASE}/auth/login`, {
			method: 'POST',
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({ email, password })
		});

		if (!response.ok) {
			const error: ApiError = {
				message: await response.text(),
				status: response.status
			};
			throw error;
		}

		const data = await response.json();
		// Store access token in memory only
		this.accessToken = data.access_token;
		return data;
	}

	async logout() {
		// Backend reads refresh token from httpOnly cookie and clears it
		await this.request('/auth/logout', { method: 'POST' });
		this.accessToken = null;
	}

	/**
	 * Refresh access token from httpOnly cookie
	 * Returns user info if successful
	 */
	async refreshToken() {
		const response = await fetch(`${API_BASE}/auth/refresh`, {
			method: 'POST',
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json'
			}
		});

		if (!response.ok) {
			throw new Error('Refresh failed');
		}

		const data = await response.json();
		this.accessToken = data.access_token;
		return data;
	}

	/**
	 * Get current client profile
	 */
	async getMe() {
		return this.request<{
			id: string;
			email: string;
			name: string;
			tenant_id: string;
			tenant_name: string;
		}>('/auth/me');
	}

	// Activation
	async validateToken(token: string) {
		return this.request<{ valid: boolean; email: string }>(`/activate/${token}`);
	}

	async activate(token: string, password: string, name: string) {
		return this.request(`/activate/${token}`, {
			method: 'POST',
			body: JSON.stringify({ password, name })
		});
	}

	// Documents (shared with client)
	async getDocuments(params?: { limit?: number; offset?: number }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		return this.request<{
			documents: any[];
			total: number;
		}>(`/documents?${query}`);
	}

	async getDocument(id: string) {
		return this.request<any>(`/documents/${id}`);
	}

	// Uploads
	async getUploads(params?: { limit?: number; offset?: number }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		return this.request<{
			uploads: any[];
			total: number;
		}>(`/uploads?${query}`);
	}

	async uploadFile(file: File, category: string, note?: string) {
		const formData = new FormData();
		formData.append('file', file);
		formData.append('category', category);
		if (note) formData.append('note', note);

		const headers: Record<string, string> = {};
		if (this.accessToken) {
			headers['Authorization'] = `Bearer ${this.accessToken}`;
		}

		const response = await fetch(`${API_BASE}/uploads`, {
			method: 'POST',
			credentials: 'include',
			headers,
			body: formData
		});

		if (!response.ok) {
			throw new Error(await response.text());
		}

		return response.json();
	}

	async deleteUpload(id: string) {
		return this.request(`/uploads/${id}`, { method: 'DELETE' });
	}

	// Approvals
	async getApprovals(params?: { status?: string; limit?: number; offset?: number }) {
		const query = new URLSearchParams();
		if (params?.status) query.set('status', params.status);
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		return this.request<{
			approvals: any[];
			total: number;
		}>(`/approvals?${query}`);
	}

	async getApproval(id: string) {
		return this.request<any>(`/approvals/${id}`);
	}

	async approveRequest(id: string) {
		return this.request(`/approvals/${id}/approve`, { method: 'POST' });
	}

	async rejectRequest(id: string, comment: string) {
		return this.request(`/approvals/${id}/reject`, {
			method: 'POST',
			body: JSON.stringify({ comment })
		});
	}

	// Tasks
	async getTasks(params?: { status?: string; limit?: number; offset?: number }) {
		const query = new URLSearchParams();
		if (params?.status) query.set('status', params.status);
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		return this.request<{
			tasks: any[];
			total: number;
		}>(`/tasks?${query}`);
	}

	async getTask(id: string) {
		return this.request<any>(`/tasks/${id}`);
	}

	async completeTask(id: string) {
		return this.request(`/tasks/${id}/complete`, { method: 'POST' });
	}

	// Messages
	async getThreads(params?: { limit?: number; offset?: number }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		return this.request<{
			threads: any[];
			total: number;
		}>(`/messages/threads?${query}`);
	}

	async getThread(id: string) {
		return this.request<any>(`/messages/threads/${id}`);
	}

	async getMessages(threadId: string, params?: { limit?: number; offset?: number }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		return this.request<{
			messages: any[];
			total: number;
		}>(`/messages/threads/${threadId}/messages?${query}`);
	}

	async startThread(subject: string, content: string) {
		return this.request<{ thread: any; message: any }>('/messages/threads', {
			method: 'POST',
			body: JSON.stringify({ subject, content })
		});
	}

	async sendMessage(threadId: string, content: string) {
		return this.request<any>(`/messages/threads/${threadId}/messages`, {
			method: 'POST',
			body: JSON.stringify({ content })
		});
	}

	async markAsRead(threadId: string) {
		return this.request(`/messages/threads/${threadId}/read`, { method: 'POST' });
	}

	async getUnreadCount() {
		return this.request<{ unread_count: number }>('/messages/unread');
	}

	// Branding
	async getBranding() {
		return this.request<any>('/branding');
	}
}

export const api = new PortalApiClient();
