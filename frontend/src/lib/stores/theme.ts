import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'system';

function getInitialTheme(): Theme {
	if (!browser) return 'system';
	const stored = localStorage.getItem('theme');
	if (stored === 'light' || stored === 'dark' || stored === 'system') {
		return stored;
	}
	return 'system';
}

function getSystemTheme(): 'light' | 'dark' {
	if (!browser) return 'light';
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function createThemeStore() {
	const { subscribe, set, update } = writable<Theme>(getInitialTheme());

	function applyTheme(theme: Theme) {
		if (!browser) return;

		const effectiveTheme = theme === 'system' ? getSystemTheme() : theme;
		const root = document.documentElement;

		if (effectiveTheme === 'dark') {
			root.classList.add('dark');
		} else {
			root.classList.remove('dark');
		}

		localStorage.setItem('theme', theme);
	}

	// Apply initial theme
	if (browser) {
		const initial = getInitialTheme();
		applyTheme(initial);

		// Listen for system theme changes
		window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
			const currentTheme = localStorage.getItem('theme') as Theme;
			if (currentTheme === 'system') {
				applyTheme('system');
			}
		});
	}

	return {
		subscribe,
		set: (theme: Theme) => {
			set(theme);
			applyTheme(theme);
		},
		toggle: () => {
			update((current) => {
				const effectiveCurrent = current === 'system' ? getSystemTheme() : current;
				const next = effectiveCurrent === 'light' ? 'dark' : 'light';
				applyTheme(next);
				return next;
			});
		},
	};
}

export const theme = createThemeStore();
