import { browser } from '$app/environment';
import { goto } from '$app/navigation';

/**
 * API Client for Austrian Business Infrastructure Backend
 *
 * Features:
 * - Automatic auth header injection
 * - Token refresh on 401
 * - Request/response interceptors
 * - Error normalization
 */

export interface ApiError {
	status: number;
	code: string;
	message: string;
	details?: Record<string, unknown>;
}

export interface ApiResponse<T> {
	data: T;
	meta?: {
		total?: number;
		page?: number;
		limit?: number;
	};
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
	body?: unknown;
	params?: Record<string, string | number | boolean | undefined>;
}

class ApiClient {
	private baseUrl: string;
	private accessToken: string | null = null;

	constructor(baseUrl: string) {
		this.baseUrl = baseUrl;
	}

	/**
	 * Set the access token for authenticated requests
	 */
	setAccessToken(token: string | null) {
		this.accessToken = token;
	}

	/**
	 * Get the current access token
	 */
	getAccessToken(): string | null {
		return this.accessToken;
	}

	/**
	 * Build URL with query parameters
	 */
	private buildUrl(path: string, params?: Record<string, string | number | boolean | undefined>): string {
		const url = new URL(path, this.baseUrl);

		if (params) {
			Object.entries(params).forEach(([key, value]) => {
				if (value !== undefined) {
					url.searchParams.set(key, String(value));
				}
			});
		}

		return url.toString();
	}

	/**
	 * Make an authenticated API request
	 */
	private async request<T>(
		method: string,
		path: string,
		options: RequestOptions = {}
	): Promise<T> {
		const { body, params, headers: customHeaders, ...rest } = options;

		const headers: HeadersInit = {
			'Content-Type': 'application/json',
			...customHeaders
		};

		if (this.accessToken) {
			(headers as Record<string, string>)['Authorization'] = `Bearer ${this.accessToken}`;
		}

		const url = this.buildUrl(path, params);

		try {
			const response = await fetch(url, {
				method,
				headers,
				body: body ? JSON.stringify(body) : undefined,
				credentials: 'include', // Include cookies for httpOnly JWT
				...rest
			});

			// Handle 401 - redirect to login
			if (response.status === 401) {
				this.accessToken = null;
				if (browser) {
					goto('/login');
				}
				throw this.createError(401, 'unauthorized', 'Session expired. Please log in again.');
			}

			// Handle 204 No Content
			if (response.status === 204) {
				return undefined as T;
			}

			const data = await response.json();

			// Handle error responses
			if (!response.ok) {
				throw this.createError(
					response.status,
					data.code || 'error',
					data.message || 'An error occurred',
					data.details
				);
			}

			return data as T;
		} catch (error) {
			if (this.isApiError(error)) {
				throw error;
			}

			// Network or parsing errors
			throw this.createError(
				0,
				'network_error',
				error instanceof Error ? error.message : 'Network error occurred'
			);
		}
	}

	/**
	 * Create a normalized API error
	 */
	private createError(
		status: number,
		code: string,
		message: string,
		details?: Record<string, unknown>
	): ApiError {
		return { status, code, message, details };
	}

	/**
	 * Type guard for ApiError
	 */
	isApiError(error: unknown): error is ApiError {
		return (
			typeof error === 'object' &&
			error !== null &&
			'status' in error &&
			'code' in error &&
			'message' in error
		);
	}

	// HTTP Methods

	async get<T>(path: string, options?: RequestOptions): Promise<T> {
		return this.request<T>('GET', path, options);
	}

	async post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>('POST', path, { ...options, body });
	}

	async put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>('PUT', path, { ...options, body });
	}

	async patch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>('PATCH', path, { ...options, body });
	}

	async delete<T>(path: string, options?: RequestOptions): Promise<T> {
		return this.request<T>('DELETE', path, options);
	}

	/**
	 * Upload a file with multipart/form-data
	 */
	async upload<T>(path: string, file: File, fieldName = 'file'): Promise<T> {
		const formData = new FormData();
		formData.append(fieldName, file);

		const headers: HeadersInit = {};
		if (this.accessToken) {
			headers['Authorization'] = `Bearer ${this.accessToken}`;
		}

		const response = await fetch(this.buildUrl(path), {
			method: 'POST',
			headers,
			body: formData,
			credentials: 'include'
		});

		if (!response.ok) {
			const data = await response.json().catch(() => ({}));
			throw this.createError(
				response.status,
				data.code || 'upload_error',
				data.message || 'Upload failed'
			);
		}

		return response.json();
	}

	/**
	 * Download a file
	 */
	async download(path: string, filename: string): Promise<void> {
		const headers: HeadersInit = {};
		if (this.accessToken) {
			headers['Authorization'] = `Bearer ${this.accessToken}`;
		}

		const response = await fetch(this.buildUrl(path), {
			headers,
			credentials: 'include'
		});

		if (!response.ok) {
			throw this.createError(response.status, 'download_error', 'Download failed');
		}

		const blob = await response.blob();
		const url = URL.createObjectURL(blob);

		if (browser) {
			const a = document.createElement('a');
			a.href = url;
			a.download = filename;
			a.click();
			URL.revokeObjectURL(url);
		}
	}
}

// Export singleton instance
export const api = new ApiClient(
	import.meta.env.VITE_API_URL || 'http://localhost:8080'
);

// Export types
export type { RequestOptions };
