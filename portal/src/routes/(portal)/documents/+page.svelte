<script lang="ts">
	import { onMount } from 'svelte';
	import { FileText, Download, Eye } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { api } from '$lib/api/client';

	let documents: any[] = [];
	let loading = true;
	let total = 0;
	let page = 0;
	const limit = 20;

	onMount(async () => {
		await loadDocuments();
	});

	async function loadDocuments() {
		loading = true;
		try {
			const result = await api.getDocuments({ limit, offset: page * limit });
			documents = result.documents || [];
			total = result.total;
		} catch (e) {
			console.error('Failed to load documents:', e);
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
</script>

<svelte:head>
	<title>Dokumente | Mandantenportal</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-gray-900">Dokumente</h1>
		<span class="text-sm text-gray-500">{total} Dokument{total !== 1 ? 'e' : ''}</span>
	</div>

	{#if loading}
		<div class="space-y-4">
			{#each [1, 2, 3, 4, 5] as _}
				<Card>
					<div class="h-16 bg-gray-100 rounded animate-pulse"></div>
				</Card>
			{/each}
		</div>
	{:else if documents.length === 0}
		<Card class="text-center py-12">
			<FileText class="w-16 h-16 mx-auto mb-4 text-gray-300" />
			<h2 class="text-xl font-semibold text-gray-900 mb-2">Keine Dokumente</h2>
			<p class="text-gray-600">
				Es wurden noch keine Dokumente mit Ihnen geteilt.
			</p>
		</Card>
	{:else}
		<div class="space-y-3">
			{#each documents as doc}
				<Card padding="none">
					<a
						href="/documents/{doc.id}"
						class="flex items-center gap-4 p-4 hover:bg-gray-50 transition-colors"
					>
						<div class="p-3 bg-gray-100 rounded-lg">
							<FileText class="w-6 h-6 text-gray-600" />
						</div>

						<div class="flex-1 min-w-0">
							<p class="font-medium text-gray-900 truncate">
								{doc.document_title || doc.file_name}
							</p>
							<p class="text-sm text-gray-500">
								Geteilt am {formatDate(doc.shared_at)}
							</p>
						</div>

						<div class="flex items-center gap-2">
							{#if doc.can_download}
								<Badge variant="success">Download</Badge>
							{/if}
							<Eye class="w-5 h-5 text-gray-400" />
						</div>
					</a>
				</Card>
			{/each}
		</div>

		<!-- Pagination -->
		{#if total > limit}
			<div class="flex justify-center gap-2">
				<button
					class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
					disabled={page === 0}
					on:click={() => { page--; loadDocuments(); }}
				>
					Zurück
				</button>
				<span class="px-4 py-2 text-sm text-gray-500">
					Seite {page + 1} von {Math.ceil(total / limit)}
				</span>
				<button
					class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
					disabled={(page + 1) * limit >= total}
					on:click={() => { page++; loadDocuments(); }}
				>
					Weiter
				</button>
			</div>
		{/if}
	{/if}
</div>
