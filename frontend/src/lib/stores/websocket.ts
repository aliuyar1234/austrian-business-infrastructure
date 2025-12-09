import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';
import { config } from '$lib/config';
import { auth } from './auth';

/**
 * WebSocket message types
 */
export type WSMessageType =
	| 'new_document'
	| 'sync_progress'
	| 'sync_complete'
	| 'deadline_reminder'
	| 'account_status'
	| 'notification';

export interface WSMessage<T = unknown> {
	type: WSMessageType;
	payload: T;
	timestamp: string;
}

export interface NewDocumentPayload {
	id: string;
	accountId: string;
	accountName: string;
	documentType: string;
	title: string;
}

export interface SyncProgressPayload {
	accountId: string;
	accountName: string;
	progress: number;
	status: 'syncing' | 'complete' | 'error';
	message?: string;
}

export interface DeadlineReminderPayload {
	accountId: string;
	accountName: string;
	deadlineType: string;
	dueDate: string;
	daysRemaining: number;
}

/**
 * WebSocket connection state
 */
interface WSState {
	isConnected: boolean;
	isConnecting: boolean;
	isAuthenticated: boolean;
	lastMessage: WSMessage | null;
	error: string | null;
}

const initialState: WSState = {
	isConnected: false,
	isConnecting: false,
	isAuthenticated: false,
	lastMessage: null,
	error: null
};

/**
 * Create WebSocket store with first-message authentication
 *
 * SECURITY: Token is sent via first message after connection, not in URL.
 * This prevents token leakage in server logs, proxy logs, and referrer headers.
 */
function createWebSocketStore() {
	const { subscribe, set, update } = writable<WSState>(initialState);

	let socket: WebSocket | null = null;
	let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
	let reconnectDelay = config.ws.reconnectDelay;
	let messageHandlers: Map<WSMessageType, Set<(payload: unknown) => void>> = new Map();

	/**
	 * Connect to WebSocket server and authenticate via first message
	 */
	function connect() {
		if (!browser) return;
		if (socket?.readyState === WebSocket.OPEN) return;

		const authState = get(auth);
		if (!authState.user) return;

		update((s) => ({ ...s, isConnecting: true, error: null }));

		try {
			// Connect WITHOUT token in URL (security improvement)
			socket = new WebSocket(config.ws.url);

			socket.onopen = () => {
				update((s) => ({
					...s,
					isConnected: true,
					isConnecting: false,
					error: null
				}));

				// Send authentication message as first message
				const token = auth.getAccessToken();
				if (token && socket?.readyState === WebSocket.OPEN) {
					socket.send(JSON.stringify({
						type: 'auth',
						token: token
					}));
				} else {
					// No token available - close connection
					socket?.close(4001, 'No authentication token');
				}
			};

			socket.onmessage = (event) => {
				try {
					const message = JSON.parse(event.data);

					// Handle auth response
					if (message.type === 'connected') {
						update((s) => ({ ...s, isAuthenticated: true }));
						reconnectDelay = config.ws.reconnectDelay;
						console.log('[WS] Authenticated and connected');
						return;
					}

					// Handle auth error
					if (message.type === 'error') {
						console.error('[WS] Error:', message.message);
						update((s) => ({ ...s, error: message.message, isAuthenticated: false }));
						if (message.code === 'invalid_token' || message.code === 'auth_timeout') {
							socket?.close(4001, 'Authentication failed');
						}
						return;
					}

					// Handle regular messages
					const wsMessage: WSMessage = message;
					update((s) => ({ ...s, lastMessage: wsMessage }));

					// Dispatch to registered handlers
					const handlers = messageHandlers.get(wsMessage.type);
					if (handlers) {
						handlers.forEach((handler) => handler(wsMessage.payload));
					}
				} catch (error) {
					console.error('[WS] Failed to parse message:', error);
				}
			};

			socket.onerror = (event) => {
				console.error('[WS] Error:', event);
				update((s) => ({ ...s, error: 'Connection error' }));
			};

			socket.onclose = (event) => {
				update((s) => ({
					...s,
					isConnected: false,
					isConnecting: false,
					isAuthenticated: false
				}));

				socket = null;

				// Auto-reconnect with exponential backoff (unless clean close or auth failure)
				if (event.code !== 1000 && event.code !== 4001) {
					scheduleReconnect();
				}
			};
		} catch (error) {
			update((s) => ({
				...s,
				isConnecting: false,
				error: 'Failed to connect'
			}));
			scheduleReconnect();
		}
	}

	/**
	 * Schedule reconnection with exponential backoff
	 */
	function scheduleReconnect() {
		if (reconnectTimeout) return;

		reconnectTimeout = setTimeout(() => {
			reconnectTimeout = null;
			reconnectDelay = Math.min(
				reconnectDelay * config.ws.reconnectMultiplier,
				config.ws.maxReconnectDelay
			);
			connect();
		}, reconnectDelay);

		console.log(`[WS] Reconnecting in ${reconnectDelay}ms`);
	}

	/**
	 * Disconnect from WebSocket server
	 */
	function disconnect() {
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
			reconnectTimeout = null;
		}

		if (socket) {
			socket.close(1000, 'User disconnect');
			socket = null;
		}

		set(initialState);
	}

	/**
	 * Send a message through WebSocket
	 */
	function send(type: string, payload: unknown) {
		const state = get({ subscribe });
		if (socket?.readyState === WebSocket.OPEN && state.isAuthenticated) {
			socket.send(JSON.stringify({ type, payload }));
		} else {
			console.warn('[WS] Cannot send - not connected or not authenticated');
		}
	}

	/**
	 * Register a message handler
	 */
	function on<T>(type: WSMessageType, handler: (payload: T) => void): () => void {
		if (!messageHandlers.has(type)) {
			messageHandlers.set(type, new Set());
		}
		messageHandlers.get(type)!.add(handler as (payload: unknown) => void);

		// Return unsubscribe function
		return () => {
			messageHandlers.get(type)?.delete(handler as (payload: unknown) => void);
		};
	}

	/**
	 * Remove all handlers for a message type
	 */
	function off(type: WSMessageType) {
		messageHandlers.delete(type);
	}

	return {
		subscribe,
		connect,
		disconnect,
		send,
		on,
		off
	};
}

export const ws = createWebSocketStore();
