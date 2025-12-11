<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { auth, isInitialized } from '$lib/stores/auth';
	import { theme } from '$lib/stores/theme';
	import CommandPalette from '$lib/components/layout/CommandPalette.svelte';
	import KeyboardShortcuts from '$lib/components/layout/KeyboardShortcuts.svelte';
	import Toast from '$lib/components/ui/Toast.svelte';

	let { children } = $props();

	// Initialize theme on mount to ensure it's applied
	let currentTheme = $state('system');
	theme.subscribe((value) => {
		currentTheme = value;
	});

	onMount(() => {
		auth.initialize();
	});
</script>

<svelte:head>
	<title>Austrian Business Infrastructure</title>
	<meta name="description" content="FinanzOnline, ELDA, Firmenbuch and more - unified Austrian business API platform" />
	<link rel="icon" type="image/svg+xml" href="/favicon.svg" />
</svelte:head>

{#if !$isInitialized}
	<!-- Loading state while auth initializes -->
	<div class="min-h-screen bg-[var(--color-paper)] flex items-center justify-center">
		<div class="flex flex-col items-center gap-4">
			<div class="w-10 h-10 rounded-lg bg-[var(--color-accent)] flex items-center justify-center animate-pulse">
				<span class="text-white font-bold text-sm">ABI</span>
			</div>
			<p class="text-sm text-[var(--color-ink-muted)]">Loading...</p>
		</div>
	</div>
{:else}
	{@render children()}
	<CommandPalette />
	<KeyboardShortcuts />
	<Toast />
{/if}
