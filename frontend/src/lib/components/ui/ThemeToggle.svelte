<script lang="ts">
	import { theme, type Theme } from '$lib/stores/theme';
	import Button from './Button.svelte';

	interface Props {
		variant?: 'icon' | 'full';
		class?: string;
	}

	let {
		variant = 'icon',
		class: className = '',
	}: Props = $props();

	let currentTheme = $state<Theme>('system');

	theme.subscribe((value) => {
		currentTheme = value;
	});

	function cycleTheme() {
		const order: Theme[] = ['light', 'dark', 'system'];
		const currentIndex = order.indexOf(currentTheme);
		const nextIndex = (currentIndex + 1) % order.length;
		theme.set(order[nextIndex]);
	}

	function getThemeLabel(t: Theme): string {
		switch (t) {
			case 'light': return 'Light';
			case 'dark': return 'Dark';
			case 'system': return 'System';
		}
	}
</script>

{#if variant === 'full'}
	<div class="flex items-center gap-2 {className}">
		<Button
			variant={currentTheme === 'light' ? 'secondary' : 'ghost'}
			size="sm"
			onclick={() => theme.set('light')}
			aria-label="Light mode"
		>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="12" cy="12" r="4"/>
				<path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
			</svg>
		</Button>
		<Button
			variant={currentTheme === 'dark' ? 'secondary' : 'ghost'}
			size="sm"
			onclick={() => theme.set('dark')}
			aria-label="Dark mode"
		>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
			</svg>
		</Button>
		<Button
			variant={currentTheme === 'system' ? 'secondary' : 'ghost'}
			size="sm"
			onclick={() => theme.set('system')}
			aria-label="System mode"
		>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect width="20" height="14" x="2" y="3" rx="2"/>
				<path d="M8 21h8M12 17v4"/>
			</svg>
		</Button>
	</div>
{:else}
	<button
		onclick={cycleTheme}
		class="p-2 rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors {className}"
		aria-label="Toggle theme ({getThemeLabel(currentTheme)})"
		title="{getThemeLabel(currentTheme)} theme"
	>
		{#if currentTheme === 'light'}
			<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="12" cy="12" r="4"/>
				<path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
			</svg>
		{:else if currentTheme === 'dark'}
			<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
			</svg>
		{:else}
			<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect width="20" height="14" x="2" y="3" rx="2"/>
				<path d="M8 21h8M12 17v4"/>
			</svg>
		{/if}
	</button>
{/if}
