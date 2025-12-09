/**
 * Application Configuration
 *
 * All configuration is loaded from environment variables
 * with sensible defaults for development
 */

export const config = {
	// API Configuration
	api: {
		baseUrl: import.meta.env.VITE_API_URL || 'http://localhost:8080',
		timeout: 30000
	},

	// WebSocket Configuration
	ws: {
		url: import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws',
		reconnectDelay: 1000,
		maxReconnectDelay: 30000,
		reconnectMultiplier: 2
	},

	// OAuth2 Providers
	oauth: {
		google: {
			clientId: import.meta.env.VITE_GOOGLE_CLIENT_ID || '',
			enabled: !!import.meta.env.VITE_GOOGLE_CLIENT_ID
		},
		microsoft: {
			clientId: import.meta.env.VITE_MICROSOFT_CLIENT_ID || '',
			enabled: !!import.meta.env.VITE_MICROSOFT_CLIENT_ID
		}
	},

	// Feature Flags
	features: {
		darkMode: import.meta.env.VITE_ENABLE_DARK_MODE !== 'false',
		notifications: import.meta.env.VITE_ENABLE_NOTIFICATIONS !== 'false'
	},

	// App Metadata
	app: {
		name: 'Austrian Business Infrastructure',
		shortName: 'ABI',
		version: '1.0.0'
	}
} as const;

export type Config = typeof config;
