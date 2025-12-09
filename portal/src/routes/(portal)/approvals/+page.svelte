<script lang="ts">
	import { onMount } from 'svelte';
	import { CheckCircle, XCircle, Clock, FileText } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { api } from '$lib/api/client';

	let approvals: any[] = [];
	let loading = true;
	let total = 0;
	let filter: 'all' | 'pending' = 'pending';

	onMount(async () => {
		await loadApprovals();
	});

	async function loadApprovals() {
		loading = true;
		try {
			const status = filter === 'pending' ? 'pending' : undefined;
			const result = await api.getApprovals({ status, limit: 50 });
			approvals = result.approvals || [];
			total = result.total;
		} catch (e) {
			console.error('Failed to load approvals:', e);
		} finally {
			loading = false;
		}
	}

	function formatDate(date: string): string {
		return new Date(date).toLocaleDateString('de-AT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		});
	}

	function getStatusBadge(status: string) {
		switch (status) {
			case 'pending':
				return { variant: 'warning' as const, label: 'Offen' };
			case 'approved':
				return { variant: 'success' as const, label: 'Freigegeben' };
			case 'rejected':
				return { variant: 'danger' as const, label: 'Abgelehnt' };
			case 'revision_requested':
				return { variant: 'info' as const, label: 'Überarbeitung' };
			default:
				return { variant: 'default' as const, label: status };
		}
	}

	$: if (filter) loadApprovals();
</script>

<svelte:head>
	<title>Freigaben | Mandantenportal</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-gray-900">Freigaben</h1>

		<div class="flex gap-2">
			<button
				class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
				class:bg-primary={filter === 'pending'}
				class:text-white={filter === 'pending'}
				class:bg-gray-100={filter !== 'pending'}
				class:text-gray-700={filter !== 'pending'}
				on:click={() => (filter = 'pending')}
			>
				Offen
			</button>
			<button
				class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
				class:bg-primary={filter === 'all'}
				class:text-white={filter === 'all'}
				class:bg-gray-100={filter !== 'all'}
				class:text-gray-700={filter !== 'all'}
				on:click={() => (filter = 'all')}
			>
				Alle
			</button>
		</div>
	</div>

	{#if loading}
		<div class="space-y-4">
			{#each [1, 2, 3] as _}
				<Card>
					<div class="h-20 bg-gray-100 rounded animate-pulse"></div>
				</Card>
			{/each}
		</div>
	{:else if approvals.length === 0}
		<Card class="text-center py-12">
			<CheckCircle class="w-16 h-16 mx-auto mb-4 text-green-500" />
			<h2 class="text-xl font-semibold text-gray-900 mb-2">
				{filter === 'pending' ? 'Keine offenen Freigaben' : 'Keine Freigaben'}
			</h2>
			<p class="text-gray-600">
				{filter === 'pending'
					? 'Alle Dokumente wurden bearbeitet.'
					: 'Es wurden noch keine Freigaben angefordert.'}
			</p>
		</Card>
	{:else}
		<div class="space-y-3">
			{#each approvals as approval}
				{@const status = getStatusBadge(approval.status)}
				<Card padding="none">
					<a
						href="/approvals/{approval.id}"
						class="flex items-center gap-4 p-4 hover:bg-gray-50 transition-colors"
					>
						<div class="p-3 rounded-lg" class:bg-yellow-100={approval.status === 'pending'} class:bg-gray-100={approval.status !== 'pending'}>
							{#if approval.status === 'pending'}
								<Clock class="w-6 h-6 text-yellow-600" />
							{:else if approval.status === 'approved'}
								<CheckCircle class="w-6 h-6 text-green-600" />
							{:else}
								<XCircle class="w-6 h-6 text-red-600" />
							{/if}
						</div>

						<div class="flex-1 min-w-0">
							<p class="font-medium text-gray-900 truncate">
								{approval.document_title}
							</p>
							<p class="text-sm text-gray-500">
								Angefordert am {formatDate(approval.requested_at)}
							</p>
							{#if approval.message}
								<p class="text-sm text-gray-600 mt-1 truncate">
									{approval.message}
								</p>
							{/if}
						</div>

						<Badge variant={status.variant}>{status.label}</Badge>
					</a>
				</Card>
			{/each}
		</div>
	{/if}
</div>
