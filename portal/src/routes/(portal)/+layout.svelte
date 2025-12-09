<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth, isAuthenticated } from '$lib/stores/auth';
	import { websocket } from '$lib/stores/websocket';
	import PortalHeader from '$lib/components/layout/PortalHeader.svelte';
	import PortalSidebar from '$lib/components/layout/PortalSidebar.svelte';

	let sidebarOpen = false;

	onMount(() => {
		// Check auth on mount
		if (!$isAuthenticated) {
			goto('/login');
			return;
		}

		// Connect WebSocket
		websocket.connect();

		return () => {
			websocket.disconnect();
		};
	});

	$: if (!$auth.loading && !$isAuthenticated) {
		goto('/login');
	}

	function toggleSidebar() {
		sidebarOpen = !sidebarOpen;
	}

	function closeSidebar() {
		sidebarOpen = false;
	}
</script>

{#if $auth.loading}
	<div class="min-h-screen flex items-center justify-center">
		<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
	</div>
{:else if $isAuthenticated}
	<div class="min-h-screen bg-gray-50">
		<PortalHeader onMenuToggle={toggleSidebar} />

		<div class="flex">
			<PortalSidebar open={sidebarOpen} onClose={closeSidebar} />

			<main class="flex-1 p-4 lg:p-6">
				<slot />
			</main>
		</div>
	</div>
{/if}
