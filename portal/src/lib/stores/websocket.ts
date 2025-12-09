import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';
import { auth } from './auth';

interface WSMessage {
	type: string;
	payload: any;
}

type MessageHandler = (payload: any) => void;

/**
 * Create WebSocket store with first-message authentication
 *
 * SECURITY: Token is sent via first message after connection, not in URL.
 * This prevents token leakage in server logs, proxy logs, and referrer headers.
 */
function createWebSocketStore() {
	const { subscribe, set, update } = writable<{
		connected: boolean;
		authenticated: boolean;
		reconnecting: boolean;
		error: string | null;
	}>({
		connected: false,
		authenticated: false,
		reconnecting: false,
		error: null
	});

	let socket: WebSocket | null = null;
	let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	let reconnectAttempts = 0;
	const maxReconnectAttempts = 5;
	const handlers = new Map<string, Set<MessageHandler>>();

	const connect = () => {
		if (!browser || socket?.readyState === WebSocket.OPEN) return;

		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		// Connect WITHOUT token in URL (security improvement)
		const wsUrl = `${protocol}//${window.location.host}/api/v1/portal/messages/ws`;

		socket = new WebSocket(wsUrl);

		socket.onopen = () => {
			update((state) => ({ ...state, connected: true, reconnecting: false, error: null }));
			reconnectAttempts = 0;

			// Send authentication message as first message
			const token = auth.getAccessToken();
			if (token && socket?.readyState === WebSocket.OPEN) {
				socket.send(JSON.stringify({
					type: 'auth',
					token: token
				}));
			} else {
				// No token available - close connection
				update((state) => ({ ...state, error: 'No authentication token' }));
				socket?.close(4001, 'No authentication token');
			}
		};

		socket.onclose = (event) => {
			set({ connected: false, authenticated: false, reconnecting: false, error: null });
			// Don't reconnect on auth failure (4001)
			if (event.code !== 4001) {
				scheduleReconnect();
			}
		};

		socket.onerror = () => {
			socket?.close();
		};

		socket.onmessage = (event) => {
			try {
				const message: WSMessage = JSON.parse(event.data);

				// Handle auth response
				if (message.type === 'connected') {
					update((state) => ({ ...state, authenticated: true }));
					console.log('[Portal WS] Authenticated and connected');
					return;
				}

				// Handle auth error
				if (message.type === 'error') {
					console.error('[Portal WS] Error:', message.payload?.message);
					update((state) => ({ ...state, error: message.payload?.message, authenticated: false }));
					if (message.payload?.code === 'invalid_token' || message.payload?.code === 'auth_timeout') {
						socket?.close(4001, 'Authentication failed');
					}
					return;
				}

				// Dispatch to registered handlers
				const typeHandlers = handlers.get(message.type);
				if (typeHandlers) {
					typeHandlers.forEach((handler) => handler(message.payload));
				}
			} catch (e) {
				console.error('Failed to parse WebSocket message:', e);
			}
		};
	};

	const scheduleReconnect = () => {
		if (reconnectAttempts >= maxReconnectAttempts) return;

		update((state) => ({ ...state, reconnecting: true }));
		reconnectAttempts++;

		const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
		reconnectTimer = setTimeout(connect, delay);
	};

	const disconnect = () => {
		if (reconnectTimer) {
			clearTimeout(reconnectTimer);
			reconnectTimer = null;
		}
		socket?.close(1000, 'User disconnect');
		socket = null;
		set({ connected: false, authenticated: false, reconnecting: false, error: null });
	};

	const send = (type: string, payload: any) => {
		const state = get({ subscribe });
		if (socket?.readyState === WebSocket.OPEN && state.authenticated) {
			socket.send(JSON.stringify({ type, payload }));
		} else {
			console.warn('[Portal WS] Cannot send - not connected or not authenticated');
		}
	};

	const on = (type: string, handler: MessageHandler) => {
		if (!handlers.has(type)) {
			handlers.set(type, new Set());
		}
		handlers.get(type)!.add(handler);

		return () => {
			handlers.get(type)?.delete(handler);
		};
	};

	return {
		subscribe,
		connect,
		disconnect,
		send,
		on
	};
}

export const websocket = createWebSocketStore();
