import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export interface Branding {
	company_name: string;
	logo_url?: string;
	favicon_url?: string;
	primary_color: string;
	secondary_color?: string;
	accent_color?: string;
	welcome_message?: string;
	footer_text?: string;
	support_email?: string;
	support_phone?: string;
}

const defaultBranding: Branding = {
	company_name: 'Client Portal',
	primary_color: '#3B82F6'
};

function createBrandingStore() {
	const { subscribe, set } = writable<Branding>(defaultBranding);

	return {
		subscribe,
		setBranding: (branding: Branding) => {
			set(branding);
			if (browser) {
				applyBrandingToDOM(branding);
			}
		},
		reset: () => {
			set(defaultBranding);
			if (browser) {
				applyBrandingToDOM(defaultBranding);
			}
		}
	};
}

function applyBrandingToDOM(branding: Branding) {
	const root = document.documentElement;

	root.style.setProperty('--primary-color', branding.primary_color);

	if (branding.secondary_color) {
		root.style.setProperty('--secondary-color', branding.secondary_color);
	}

	if (branding.accent_color) {
		root.style.setProperty('--accent-color', branding.accent_color);
	}

	// Update page title
	document.title = branding.company_name;

	// Update favicon if provided
	if (branding.favicon_url) {
		const favicon = document.querySelector('link[rel="icon"]') as HTMLLinkElement;
		if (favicon) {
			favicon.href = branding.favicon_url;
		}
	}
}

export const branding = createBrandingStore();
