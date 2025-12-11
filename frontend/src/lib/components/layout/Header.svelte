<script lang="ts">
	import { cn } from '$lib/utils';
	import { user, auth } from '$lib/stores/auth';
	import { commandPalette } from '$lib/stores/commandPalette';
	import Button from '$lib/components/ui/Button.svelte';
	import ThemeToggle from '$lib/components/ui/ThemeToggle.svelte';

	interface Props {
		title?: string;
		subtitle?: string;
	}

	let { title, subtitle }: Props = $props();

	let showUserMenu = $state(false);

	function handleKeydown(e: KeyboardEvent) {
		// Escape to close menus
		if (e.key === 'Escape') {
			showUserMenu = false;
		}
	}

	async function handleLogout() {
		await auth.logout();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<header class="h-16 bg-[var(--color-paper-elevated)] border-b border-black/6 flex items-center justify-between px-6">
	<!-- Left: Title or breadcrumb -->
	<div class="flex items-center gap-4">
		{#if title}
			<div>
				<h1 class="text-lg font-semibold text-[var(--color-ink)]">{title}</h1>
				{#if subtitle}
					<p class="text-sm text-[var(--color-ink-muted)]">{subtitle}</p>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Right: Search and user menu -->
	<div class="flex items-center gap-3">
		<!-- Search button -->
		<button
			onclick={() => commandPalette.open()}
			class="
				flex items-center gap-2 h-9 px-3
				bg-[var(--color-paper-inset)] rounded-lg
				text-sm text-[var(--color-ink-muted)]
				border border-transparent
				hover:border-black/10 hover:text-[var(--color-ink-secondary)]
				transition-all duration-150
			"
		>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>
			</svg>
			<span class="hidden sm:inline">Search...</span>
			<kbd class="hidden sm:inline ml-2 px-1.5 py-0.5 text-[10px] font-medium bg-[var(--color-paper-elevated)] rounded border border-black/10">
				⌘K
			</kbd>
		</button>

		<!-- Theme toggle -->
		<ThemeToggle />

		<!-- Notifications -->
		<button
			class="
				relative w-9 h-9 flex items-center justify-center rounded-lg
				text-[var(--color-ink-muted)]
				hover:bg-[var(--color-paper-inset)] hover:text-[var(--color-ink-secondary)]
				transition-colors
			"
			aria-label="Notifications"
		>
			<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/>
				<path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>
			</svg>
			<!-- Notification dot -->
			<span class="absolute top-2 right-2 w-2 h-2 rounded-full bg-[var(--color-accent)]"></span>
		</button>

		<!-- User menu -->
		<div class="relative">
			<button
				onclick={() => showUserMenu = !showUserMenu}
				class="
					flex items-center gap-2 h-9 pl-2 pr-3 rounded-lg
					hover:bg-[var(--color-paper-inset)]
					transition-colors
				"
			>
				{#if $user}
					<div class="w-7 h-7 rounded-full bg-[var(--color-accent-muted)] flex items-center justify-center">
						<span class="text-xs font-medium text-[var(--color-accent)]">
							{$user.name.split(' ').map(n => n[0]).join('').slice(0, 2).toUpperCase()}
						</span>
					</div>
					<svg class={cn("w-4 h-4 text-[var(--color-ink-muted)] transition-transform", showUserMenu && "rotate-180")} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="m6 9 6 6 6-6"/>
					</svg>
				{/if}
			</button>

			<!-- Dropdown -->
			{#if showUserMenu}
				<div
					class="
						absolute right-0 top-full mt-1
						w-56 py-1
						bg-[var(--color-paper-elevated)]
						border border-black/10 rounded-lg
						shadow-lg
						animate-in
						z-50
					"
				>
					{#if $user}
						<div class="px-3 py-2 border-b border-black/6">
							<p class="text-sm font-medium text-[var(--color-ink)]">{$user.name}</p>
							<p class="text-xs text-[var(--color-ink-muted)]">{$user.email}</p>
						</div>
					{/if}

					<a
						href="/settings"
						class="flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-ink-secondary)] hover:bg-[var(--color-paper-inset)]"
						onclick={() => showUserMenu = false}
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
							<circle cx="12" cy="12" r="3"/>
						</svg>
						Settings
					</a>

					<button
						onclick={handleLogout}
						class="w-full flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-error)] hover:bg-[var(--color-error-muted)]"
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
							<polyline points="16 17 21 12 16 7"/>
							<line x1="21" x2="9" y1="12" y2="12"/>
						</svg>
						Sign out
					</button>
				</div>
			{/if}
		</div>
	</div>
</header>

<!-- Click outside to close menu -->
{#if showUserMenu}
	<button
		class="fixed inset-0 z-40"
		onclick={() => showUserMenu = false}
		aria-label="Close menu"
	></button>
{/if}
