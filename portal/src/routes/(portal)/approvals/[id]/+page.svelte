<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { ArrowLeft, FileText, CheckCircle, XCircle } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { api } from '$lib/api/client';

	let approval: any = null;
	let loading = true;
	let error = '';
	let processing = false;
	let showRejectModal = false;
	let rejectComment = '';

	onMount(async () => {
		const id = $page.params.id;

		try {
			approval = await api.getApproval(id);
		} catch (e: any) {
			error = e.message || 'Freigabe nicht gefunden';
		} finally {
			loading = false;
		}
	});

	async function handleApprove() {
		processing = true;
		try {
			await api.approveRequest(approval.id);
			approval.status = 'approved';
		} catch (e: any) {
			error = e.message || 'Freigabe fehlgeschlagen';
		} finally {
			processing = false;
		}
	}

	async function handleReject() {
		if (!rejectComment.trim()) {
			error = 'Bitte geben Sie einen Grund an';
			return;
		}

		processing = true;
		try {
			await api.rejectRequest(approval.id, rejectComment);
			approval.status = 'rejected';
			showRejectModal = false;
		} catch (e: any) {
			error = e.message || 'Ablehnung fehlgeschlagen';
		} finally {
			processing = false;
		}
	}

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
	<title>Freigabe | Mandantenportal</title>
</svelte:head>

<div class="space-y-6">
	<button
		class="inline-flex items-center gap-2 text-gray-600 hover:text-gray-900"
		on:click={() => goto('/approvals')}
	>
		<ArrowLeft class="w-4 h-4" />
		Zurück zu Freigaben
	</button>

	{#if loading}
		<Card>
			<div class="h-96 bg-gray-100 rounded animate-pulse"></div>
		</Card>
	{:else if error && !approval}
		<Card class="text-center py-12">
			<p class="text-red-600">{error}</p>
		</Card>
	{:else if approval}
		<Card>
			<div class="flex items-start justify-between mb-6">
				<div class="flex items-center gap-4">
					<div class="p-3 bg-gray-100 rounded-lg">
						<FileText class="w-8 h-8 text-gray-600" />
					</div>
					<div>
						<h1 class="text-xl font-bold text-gray-900">
							{approval.document_title}
						</h1>
						<p class="text-sm text-gray-500">
							Angefordert am {formatDate(approval.requested_at)}
						</p>
					</div>
				</div>

				{#if approval.status === 'pending'}
					<Badge variant="warning">Offen</Badge>
				{:else if approval.status === 'approved'}
					<Badge variant="success">Freigegeben</Badge>
				{:else if approval.status === 'rejected'}
					<Badge variant="danger">Abgelehnt</Badge>
				{/if}
			</div>

			{#if approval.message}
				<div class="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
					<p class="text-sm font-medium text-blue-800 mb-1">Nachricht:</p>
					<p class="text-blue-700">{approval.message}</p>
				</div>
			{/if}

			{#if error}
				<div class="mb-6 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
					{error}
				</div>
			{/if}

			<!-- Document preview placeholder -->
			<div class="border border-gray-200 rounded-lg bg-gray-50 min-h-[400px] flex items-center justify-center mb-6">
				<div class="text-center text-gray-500">
					<FileText class="w-16 h-16 mx-auto mb-4" />
					<p>Dokumentvorschau</p>
				</div>
			</div>

			{#if approval.status === 'pending'}
				<div class="flex gap-4">
					<Button
						class="flex-1"
						variant="primary"
						loading={processing}
						on:click={handleApprove}
					>
						<CheckCircle class="w-4 h-4 mr-2" />
						Freigeben
					</Button>
					<Button
						class="flex-1"
						variant="danger"
						loading={processing}
						on:click={() => (showRejectModal = true)}
					>
						<XCircle class="w-4 h-4 mr-2" />
						Ablehnen
					</Button>
				</div>
			{:else if approval.response_comment}
				<div class="p-4 bg-gray-50 border border-gray-200 rounded-lg">
					<p class="text-sm font-medium text-gray-700 mb-1">Ihre Antwort:</p>
					<p class="text-gray-600">{approval.response_comment}</p>
					{#if approval.responded_at}
						<p class="text-xs text-gray-500 mt-2">
							{formatDate(approval.responded_at)}
						</p>
					{/if}
				</div>
			{/if}
		</Card>
	{/if}
</div>

<!-- Reject Modal -->
{#if showRejectModal}
	<div
		class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4"
		on:click={() => (showRejectModal = false)}
		on:keydown={(e) => e.key === 'Escape' && (showRejectModal = false)}
		role="dialog"
	>
		<div
			class="bg-white rounded-lg shadow-xl max-w-md w-full p-6"
			on:click|stopPropagation
			role="document"
		>
			<h2 class="text-xl font-bold text-gray-900 mb-4">Dokument ablehnen</h2>

			<p class="text-gray-600 mb-4">
				Bitte geben Sie einen Grund für die Ablehnung an:
			</p>

			<textarea
				bind:value={rejectComment}
				rows="4"
				placeholder="Grund der Ablehnung..."
				class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary resize-none mb-4"
			></textarea>

			<div class="flex gap-3">
				<Button
					variant="outline"
					class="flex-1"
					on:click={() => (showRejectModal = false)}
				>
					Abbrechen
				</Button>
				<Button
					variant="danger"
					class="flex-1"
					loading={processing}
					on:click={handleReject}
				>
					Ablehnen
				</Button>
			</div>
		</div>
	</div>
{/if}
