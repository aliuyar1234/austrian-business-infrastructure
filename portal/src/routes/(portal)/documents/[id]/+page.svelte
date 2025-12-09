<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { ArrowLeft, Download, FileText } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { api } from '$lib/api/client';

	let document: any = null;
	let loading = true;
	let error = '';

	onMount(async () => {
		const id = $page.params.id;

		try {
			document = await api.getDocument(id);
		} catch (e: any) {
			error = e.message || 'Dokument nicht gefunden';
		} finally {
			loading = false;
		}
	});

	function formatDate(date: string): string {
		return new Date(date).toLocaleDateString('de-AT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<svelte:head>
	<title>{document?.document_title || 'Dokument'} | Mandantenportal</title>
</svelte:head>

<div class="space-y-6">
	<button
		class="inline-flex items-center gap-2 text-gray-600 hover:text-gray-900"
		on:click={() => goto('/documents')}
	>
		<ArrowLeft class="w-4 h-4" />
		Zurück zu Dokumente
	</button>

	{#if loading}
		<Card>
			<div class="h-96 bg-gray-100 rounded animate-pulse"></div>
		</Card>
	{:else if error}
		<Card class="text-center py-12">
			<p class="text-red-600">{error}</p>
		</Card>
	{:else if document}
		<Card>
			<div class="flex items-start justify-between mb-6">
				<div class="flex items-center gap-4">
					<div class="p-3 bg-gray-100 rounded-lg">
						<FileText class="w-8 h-8 text-gray-600" />
					</div>
					<div>
						<h1 class="text-xl font-bold text-gray-900">
							{document.document_title}
						</h1>
						<p class="text-sm text-gray-500">
							Geteilt am {formatDate(document.shared_at)}
						</p>
					</div>
				</div>

				{#if document.can_download}
					<Button variant="outline">
						<Download class="w-4 h-4 mr-2" />
						Download
					</Button>
				{/if}
			</div>

			<!-- Document preview placeholder -->
			<div class="border border-gray-200 rounded-lg bg-gray-50 min-h-[500px] flex items-center justify-center">
				<div class="text-center text-gray-500">
					<FileText class="w-16 h-16 mx-auto mb-4" />
					<p>Dokumentvorschau</p>
					<p class="text-sm">PDF-Viewer würde hier angezeigt</p>
				</div>
			</div>
		</Card>
	{/if}
</div>
