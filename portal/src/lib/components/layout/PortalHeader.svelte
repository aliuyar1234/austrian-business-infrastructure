<script lang="ts">
	import { Menu, Bell, LogOut, User, Settings } from 'lucide-svelte';
	import { auth } from '$lib/stores/auth';
	import { branding } from '$lib/stores/branding';
	import { api } from '$lib/api/client';
	import { goto } from '$app/navigation';

	export let onMenuToggle: () => void;

	let showUserMenu = false;
	let unreadCount = 0;

	async function handleLogout() {
		try {
			await api.logout();
		} catch {
			// Ignore errors on logout
		}
		auth.logout();
		goto('/login');
	}

	function toggleUserMenu() {
		showUserMenu = !showUserMenu;
	}

	// Close menu when clicking outside
	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.user-menu')) {
			showUserMenu = false;
		}
	}
</script>

<svelte:window on:click={handleClickOutside} />

<header class="bg-white border-b border-gray-200 px-4 py-3">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<button
				class="lg:hidden p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
				on:click={onMenuToggle}
			>
				<Menu class="w-5 h-5" />
			</button>

			{#if $branding.logo_url}
				<img src={$branding.logo_url} alt={$branding.company_name} class="h-8" />
			{:else}
				<span class="text-xl font-semibold text-gray-900">
					{$branding.company_name}
				</span>
			{/if}
		</div>

		<div class="flex items-center gap-3">
			<a
				href="/messages"
				class="relative p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
			>
				<Bell class="w-5 h-5" />
				{#if unreadCount > 0}
					<span
						class="absolute top-0 right-0 inline-flex items-center justify-center w-4 h-4 text-xs font-bold text-white bg-red-500 rounded-full"
					>
						{unreadCount > 9 ? '9+' : unreadCount}
					</span>
				{/if}
			</a>

			<div class="relative user-menu">
				<button
					class="flex items-center gap-2 p-2 text-gray-700 hover:bg-gray-100 rounded-lg"
					on:click|stopPropagation={toggleUserMenu}
				>
					<div class="w-8 h-8 bg-primary/10 text-primary rounded-full flex items-center justify-center">
						<User class="w-4 h-4" />
					</div>
					<span class="hidden sm:block text-sm font-medium">
						{$auth.client?.name || 'User'}
					</span>
				</button>

				{#if showUserMenu}
					<div
						class="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50"
					>
						<a
							href="/settings"
							class="flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
							on:click={() => (showUserMenu = false)}
						>
							<Settings class="w-4 h-4" />
							Einstellungen
						</a>
						<hr class="my-1 border-gray-200" />
						<button
							class="flex items-center gap-2 w-full px-4 py-2 text-sm text-red-600 hover:bg-red-50"
							on:click={handleLogout}
						>
							<LogOut class="w-4 h-4" />
							Abmelden
						</button>
					</div>
				{/if}
			</div>
		</div>
	</div>
</header>
