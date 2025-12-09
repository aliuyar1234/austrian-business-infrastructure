<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { isAuthenticated } from '$lib/stores/auth';
	import { ws } from '$lib/stores/websocket';
	import Sidebar from '$lib/components/layout/Sidebar.svelte';
	import Header from '$lib/components/layout/Header.svelte';

	let { children } = $props();

	// Route guard - redirect to login if not authenticated
	$effect(() => {
		if (!$isAuthenticated) {
			goto('/login');
		}
	});

	// Connect WebSocket when authenticated
	onMount(() => {
		if ($isAuthenticated) {
			ws.connect();
		}

		return () => {
			ws.disconnect();
		};
	});

	// Page titles based on route
	const pageTitles: Record<string, { title: string; subtitle?: string }> = {
		'/': { title: 'Dashboard', subtitle: 'Overview of your accounts and documents' },
		'/documents': { title: 'Documents', subtitle: 'View and manage your documents' },
		'/accounts': { title: 'Accounts', subtitle: 'Manage your FinanzOnline accounts' },
		'/uva': { title: 'UVA', subtitle: 'Create and submit VAT returns' },
		'/invoices': { title: 'E-Rechnung', subtitle: 'Create electronic invoices' },
		'/sepa': { title: 'SEPA', subtitle: 'Manage payments and direct debits' },
		'/firmenbuch': { title: 'Firmenbuch', subtitle: 'Search the Austrian company register' },
		'/calendar': { title: 'Calendar', subtitle: 'Upcoming deadlines and events' },
		'/team': { title: 'Team', subtitle: 'Manage your team members' },
		'/settings': { title: 'Settings', subtitle: 'Account and application settings' }
	};

	let pageInfo = $derived(pageTitles[$page.url.pathname] || { title: 'Dashboard' });
</script>

{#if $isAuthenticated}
	<div class="min-h-screen bg-[var(--color-paper)] flex">
		<!-- Sidebar -->
		<Sidebar />

		<!-- Main content -->
		<div class="flex-1 flex flex-col min-w-0">
			<Header title={pageInfo.title} subtitle={pageInfo.subtitle} />

			<main class="flex-1 overflow-auto p-6">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
