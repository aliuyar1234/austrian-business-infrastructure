<script lang="ts">
	import { goto } from '$app/navigation';
	import { formatDate } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	interface Document {
		id: string;
		name: string;
		type: string;
		size: number;
		createdAt: Date;
		accountName?: string;
	}

	// Mock documents
	let documents = $state<Document[]>([
		{ id: '1', name: 'Lohnzettel_Januar_2024_Mustermann.pdf', type: 'application/pdf', size: 125000, createdAt: new Date(), accountName: 'Muster GmbH' },
		{ id: '2', name: 'Lohnzettel_Januar_2024_Schmidt.pdf', type: 'application/pdf', size: 130000, createdAt: new Date(), accountName: 'Muster GmbH' },
		{ id: '3', name: 'Lohnzettel_Januar_2024_Mueller.pdf', type: 'application/pdf', size: 128000, createdAt: new Date(), accountName: 'Muster GmbH' },
		{ id: '4', name: 'Lohnzettel_Januar_2024_Huber.pdf', type: 'application/pdf', size: 132000, createdAt: new Date(), accountName: 'Muster GmbH' },
		{ id: '5', name: 'Lohnzettel_Januar_2024_Wagner.pdf', type: 'application/pdf', size: 127000, createdAt: new Date(), accountName: 'Muster GmbH' },
		{ id: '6', name: 'Vertrag_ABC_GmbH.pdf', type: 'application/pdf', size: 250000, createdAt: new Date(Date.now() - 86400000), accountName: 'ABC GmbH' },
		{ id: '7', name: 'Vertrag_XYZ_AG.pdf', type: 'application/pdf', size: 245000, createdAt: new Date(Date.now() - 86400000), accountName: 'XYZ AG' },
	]);

	let selectedIds = $state<Set<string>>(new Set());
	let searchQuery = $state('');
	let isSubmitting = $state(false);
	let reason = $state('');

	let filteredDocuments = $derived(
		documents.filter(doc =>
			!searchQuery ||
			doc.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			doc.accountName?.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	let allSelected = $derived(
		filteredDocuments.length > 0 && filteredDocuments.every(doc => selectedIds.has(doc.id))
	);

	function toggleAll() {
		if (allSelected) {
			selectedIds = new Set();
		} else {
			selectedIds = new Set(filteredDocuments.map(doc => doc.id));
		}
	}

	function toggleSelection(id: string) {
		const newSet = new Set(selectedIds);
		if (newSet.has(id)) {
			newSet.delete(id);
		} else {
			newSet.add(id);
		}
		selectedIds = newSet;
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
	}

	async function startBatchSigning() {
		if (selectedIds.size === 0 || isSubmitting) return;

		isSubmitting = true;
		try {
			// TODO: Call API to create batch
			// const response = await fetch('/api/v1/signatures/batch', {
			//   method: 'POST',
			//   headers: { 'Content-Type': 'application/json' },
			//   body: JSON.stringify({
			//     document_ids: Array.from(selectedIds),
			//     reason: reason
			//   })
			// });
			// const { batch_id } = await response.json();

			// Simulate
			await new Promise(resolve => setTimeout(resolve, 1000));

			// Redirect to batch signing flow
			goto('/signatures/batch/sign?batch_id=mock-batch-id');
		} catch (error) {
			console.error('Failed to create batch:', error);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="max-w-5xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<Button variant="ghost" href="/signatures">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m15 18-6-6 6-6"/>
			</svg>
		</Button>
		<div>
			<h1 class="text-2xl font-semibold text-[var(--color-ink)]">Massensignatur</h1>
			<p class="text-[var(--color-ink-muted)]">Mehrere Dokumente mit einer Authentifizierung signieren</p>
		</div>
	</div>

	<!-- Selection info -->
	{#if selectedIds.size > 0}
		<div class="bg-[var(--color-accent-muted)] border border-[var(--color-accent)] rounded-lg p-4">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<div class="w-10 h-10 bg-[var(--color-accent)] rounded-full flex items-center justify-center">
						<span class="text-white font-bold">{selectedIds.size}</span>
					</div>
					<div>
						<p class="font-medium text-[var(--color-ink)]">{selectedIds.size} Dokument{selectedIds.size !== 1 ? 'e' : ''} ausgewaehlt</p>
						<p class="text-sm text-[var(--color-ink-muted)]">Maximal 100 Dokumente pro Batch</p>
					</div>
				</div>
				<Button onclick={() => selectedIds = new Set()} variant="secondary" size="sm">
					Auswahl aufheben
				</Button>
			</div>
		</div>
	{/if}

	<!-- Document selection -->
	<Card>
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-lg font-medium text-[var(--color-ink)]">Dokumente auswaehlen</h2>
			<div class="w-64">
				<Input
					type="search"
					bind:value={searchQuery}
					placeholder="Dokumente suchen..."
				/>
			</div>
		</div>

		<div class="border border-[var(--color-border)] rounded-lg overflow-hidden">
			<table class="w-full">
				<thead class="bg-[var(--color-paper-inset)]">
					<tr>
						<th class="w-12 px-4 py-3">
							<input
								type="checkbox"
								checked={allSelected}
								onchange={toggleAll}
								class="rounded border-[var(--color-border)]"
							/>
						</th>
						<th class="text-left px-4 py-3 text-sm font-medium text-[var(--color-ink-muted)]">Dokument</th>
						<th class="text-left px-4 py-3 text-sm font-medium text-[var(--color-ink-muted)]">Konto</th>
						<th class="text-left px-4 py-3 text-sm font-medium text-[var(--color-ink-muted)]">Groesse</th>
						<th class="text-left px-4 py-3 text-sm font-medium text-[var(--color-ink-muted)]">Datum</th>
					</tr>
				</thead>
				<tbody>
					{#each filteredDocuments as doc}
						<tr
							class="border-t border-[var(--color-border)] hover:bg-[var(--color-paper-inset)] cursor-pointer {selectedIds.has(doc.id) ? 'bg-[var(--color-accent-muted)]/30' : ''}"
							onclick={() => toggleSelection(doc.id)}
						>
							<td class="px-4 py-3">
								<input
									type="checkbox"
									checked={selectedIds.has(doc.id)}
									onchange={(e) => { e.stopPropagation(); toggleSelection(doc.id); }}
									class="rounded border-[var(--color-border)]"
								/>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-3">
									<div class="w-8 h-8 bg-[var(--color-paper-inset)] rounded flex items-center justify-center flex-shrink-0">
										<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
											<polyline points="14 2 14 8 20 8"/>
										</svg>
									</div>
									<span class="text-sm font-medium text-[var(--color-ink)] truncate max-w-[300px]">{doc.name}</span>
								</div>
							</td>
							<td class="px-4 py-3">
								<span class="text-sm text-[var(--color-ink-secondary)]">{doc.accountName || '—'}</span>
							</td>
							<td class="px-4 py-3">
								<span class="text-sm text-[var(--color-ink-muted)]">{formatSize(doc.size)}</span>
							</td>
							<td class="px-4 py-3">
								<span class="text-sm text-[var(--color-ink-muted)]">{formatDate(doc.createdAt)}</span>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>

			{#if filteredDocuments.length === 0}
				<div class="p-8 text-center">
					<p class="text-[var(--color-ink-muted)]">Keine Dokumente gefunden</p>
				</div>
			{/if}
		</div>
	</Card>

	<!-- Signature settings -->
	{#if selectedIds.size > 0}
		<Card>
			<h2 class="text-lg font-medium text-[var(--color-ink)] mb-4">Signatureinstellungen</h2>

			<div class="space-y-4">
				<div>
					<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Signaturgrund (optional)</label>
					<Input bind:value={reason} placeholder="z.B. Genehmigung, Freigabe" />
					<p class="mt-1 text-xs text-[var(--color-ink-muted)]">Dieser Grund wird bei allen Dokumenten verwendet</p>
				</div>
			</div>
		</Card>

		<!-- Action -->
		<div class="flex justify-end gap-3">
			<Button variant="secondary" href="/signatures">Abbrechen</Button>
			<Button onclick={startBatchSigning} disabled={isSubmitting}>
				{#if isSubmitting}
					<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 12a9 9 0 1 1-6.219-8.56"/>
					</svg>
					Batch wird erstellt...
				{:else}
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
					</svg>
					{selectedIds.size} Dokumente signieren
				{/if}
			</Button>
		</div>
	{/if}

	<!-- Info box -->
	<div class="bg-[var(--color-paper-inset)] rounded-lg p-4">
		<div class="flex gap-3">
			<svg class="w-5 h-5 text-[var(--color-ink-muted)] flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="12" cy="12" r="10"/>
				<path d="M12 16v-4"/>
				<path d="M12 8h.01"/>
			</svg>
			<div class="text-sm text-[var(--color-ink-secondary)]">
				<p class="font-medium">So funktioniert die Massensignatur</p>
				<ol class="mt-2 list-decimal list-inside space-y-1">
					<li>Waehlen Sie die zu signierenden Dokumente aus</li>
					<li>Authentifizieren Sie sich einmalig mit ID Austria</li>
					<li>Alle ausgewaehlten Dokumente werden automatisch signiert</li>
					<li>Der Fortschritt wird in Echtzeit angezeigt</li>
				</ol>
			</div>
		</div>
	</div>
</div>
