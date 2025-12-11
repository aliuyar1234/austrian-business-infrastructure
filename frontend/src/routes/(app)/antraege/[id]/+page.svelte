<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import {
		ArrowLeft, FileText, Clock, CheckCircle, XCircle, AlertCircle,
		Euro, Calendar, Paperclip, Plus, ChevronRight, Save, Send
	} from 'lucide-svelte';

	interface TimelineEntry {
		date: string;
		status: string;
		description: string;
	}

	interface Attachment {
		name: string;
		type: string;
		url: string;
		uploaded_at: string;
	}

	interface Antrag {
		id: string;
		profile_id: string;
		foerderung_id: string;
		status: string;
		internal_reference?: string;
		submitted_at?: string;
		requested_amount?: number;
		approved_amount?: number;
		decision_date?: string;
		decision_notes?: string;
		attachments: Attachment[];
		timeline: TimelineEntry[];
		notes?: string;
		created_at: string;
		updated_at: string;
	}

	let antrag: Antrag | null = null;
	let loading = true;
	let saving = false;
	let error = '';

	// Edit fields
	let internalReference = '';
	let requestedAmount: number | null = null;
	let notes = '';
	let newStatus = '';
	let statusDescription = '';

	$: antragId = $page.params.id;

	const statusLabels: Record<string, { label: string; color: string; icon: typeof Clock }> = {
		'planned': { label: 'Geplant', color: 'bg-gray-100 text-gray-700', icon: Clock },
		'drafting': { label: 'In Bearbeitung', color: 'bg-blue-100 text-blue-700', icon: FileText },
		'submitted': { label: 'Eingereicht', color: 'bg-amber-100 text-amber-700', icon: Clock },
		'in_review': { label: 'In Prüfung', color: 'bg-purple-100 text-purple-700', icon: AlertCircle },
		'approved': { label: 'Bewilligt', color: 'bg-green-100 text-green-700', icon: CheckCircle },
		'rejected': { label: 'Abgelehnt', color: 'bg-red-100 text-red-700', icon: XCircle },
		'withdrawn': { label: 'Zurückgezogen', color: 'bg-gray-100 text-gray-500', icon: XCircle }
	};

	const statusTransitions: Record<string, string[]> = {
		'planned': ['drafting', 'withdrawn'],
		'drafting': ['planned', 'submitted', 'withdrawn'],
		'submitted': ['in_review', 'withdrawn'],
		'in_review': ['approved', 'rejected', 'withdrawn'],
		'approved': [],
		'rejected': [],
		'withdrawn': []
	};

	async function loadAntrag() {
		loading = true;
		error = '';
		try {
			antrag = await api.get<Antrag>(`/api/v1/antraege/${antragId}`);
			if (antrag) {
				internalReference = antrag.internal_reference || '';
				requestedAmount = antrag.requested_amount || null;
				notes = antrag.notes || '';
			}
		} catch (e) {
			error = 'Antrag nicht gefunden';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	async function saveAntrag() {
		if (!antrag) return;
		saving = true;
		error = '';
		try {
			await api.put(`/api/v1/antraege/${antrag.id}`, {
				internal_reference: internalReference || undefined,
				requested_amount: requestedAmount || undefined,
				notes: notes || undefined
			});
			await loadAntrag();
		} catch (e) {
			error = 'Fehler beim Speichern';
			console.error(e);
		} finally {
			saving = false;
		}
	}

	async function updateStatus() {
		if (!antrag || !newStatus) return;
		saving = true;
		error = '';
		try {
			await api.post(`/api/v1/antraege/${antrag.id}/status`, {
				status: newStatus,
				description: statusDescription || `Status geändert zu ${statusLabels[newStatus]?.label}`
			});
			newStatus = '';
			statusDescription = '';
			await loadAntrag();
		} catch (e) {
			error = 'Fehler beim Aktualisieren des Status';
			console.error(e);
		} finally {
			saving = false;
		}
	}

	function formatAmount(amount?: number): string {
		if (!amount) return '-';
		return new Intl.NumberFormat('de-AT', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(amount);
	}

	onMount(loadAntrag);
</script>

<svelte:head>
	<title>Antrag | Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<a href="/antraege" class="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mb-6">
		<ArrowLeft class="w-4 h-4" />
		Zurück zur Übersicht
	</a>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if error && !antrag}
		<div class="alert alert-error">{error}</div>
	{:else if antrag}
		{@const status = statusLabels[antrag.status] || statusLabels['planned']}
		{@const availableTransitions = statusTransitions[antrag.status] || []}

		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
			<!-- Main Content -->
			<div class="lg:col-span-2 space-y-6">
				<!-- Header Card -->
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
					<div class="flex items-center justify-between mb-4">
						<div class="flex items-center gap-2">
							<span class="px-3 py-1 text-sm font-medium rounded-full {status.color} flex items-center gap-1">
								<svelte:component this={status.icon} class="w-4 h-4" />
								{status.label}
							</span>
						</div>
						{#if availableTransitions.length > 0}
							<div class="flex items-center gap-2">
								<select bind:value={newStatus} class="select select-bordered select-sm">
									<option value="">Status ändern...</option>
									{#each availableTransitions as trans}
										<option value={trans}>{statusLabels[trans]?.label}</option>
									{/each}
								</select>
								{#if newStatus}
									<button on:click={updateStatus} disabled={saving} class="btn btn-primary btn-sm">
										{#if saving}
											<span class="loading loading-spinner loading-xs"></span>
										{:else}
											<ChevronRight class="w-4 h-4" />
										{/if}
									</button>
								{/if}
							</div>
						{/if}
					</div>

					<h1 class="text-xl font-bold text-gray-900 mb-2">
						Antrag {antrag.internal_reference || antrag.id.slice(0, 8)}
					</h1>

					<div class="grid grid-cols-2 gap-4 mt-4">
						<div>
							<div class="text-sm text-gray-500">Beantragt</div>
							<div class="text-lg font-semibold">{formatAmount(antrag.requested_amount)}</div>
						</div>
						<div>
							<div class="text-sm text-gray-500">Bewilligt</div>
							<div class="text-lg font-semibold text-green-600">{formatAmount(antrag.approved_amount)}</div>
						</div>
						{#if antrag.submitted_at}
							<div>
								<div class="text-sm text-gray-500">Eingereicht am</div>
								<div class="font-medium">{new Date(antrag.submitted_at).toLocaleDateString('de-AT')}</div>
							</div>
						{/if}
						{#if antrag.decision_date}
							<div>
								<div class="text-sm text-gray-500">Entscheidung am</div>
								<div class="font-medium">{new Date(antrag.decision_date).toLocaleDateString('de-AT')}</div>
							</div>
						{/if}
					</div>
				</div>

				<!-- Edit Form -->
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
					<h2 class="text-lg font-semibold text-gray-900 mb-4">Antragsdetails</h2>

					{#if error}
						<div class="alert alert-error mb-4">{error}</div>
					{/if}

					<div class="space-y-4">
						<div>
							<label class="label">Interne Referenz</label>
							<input type="text" bind:value={internalReference} class="input input-bordered w-full" placeholder="z.B. FOERD-2024-001" />
						</div>
						<div>
							<label class="label">Beantragter Betrag (€)</label>
							<input type="number" bind:value={requestedAmount} class="input input-bordered w-full" min="0" placeholder="z.B. 50000" />
						</div>
						<div>
							<label class="label">Notizen</label>
							<textarea bind:value={notes} class="textarea textarea-bordered w-full h-24" placeholder="Interne Notizen zum Antrag..."></textarea>
						</div>
						<div class="flex justify-end">
							<button on:click={saveAntrag} disabled={saving} class="btn btn-primary">
								{#if saving}
									<span class="loading loading-spinner loading-sm"></span>
								{:else}
									<Save class="w-4 h-4 mr-2" />
								{/if}
								Speichern
							</button>
						</div>
					</div>
				</div>

				<!-- Decision Notes -->
				{#if antrag.decision_notes}
					<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
						<h2 class="text-lg font-semibold text-gray-900 mb-4">Entscheidungsbegründung</h2>
						<p class="text-gray-700">{antrag.decision_notes}</p>
					</div>
				{/if}

				<!-- Attachments -->
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
					<div class="flex items-center justify-between mb-4">
						<h2 class="text-lg font-semibold text-gray-900">Dokumente</h2>
						<button class="btn btn-outline btn-sm">
							<Plus class="w-4 h-4 mr-1" />
							Hochladen
						</button>
					</div>

					{#if antrag.attachments && antrag.attachments.length > 0}
						<div class="space-y-2">
							{#each antrag.attachments as attachment}
								<div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
									<div class="flex items-center gap-3">
										<Paperclip class="w-4 h-4 text-gray-400" />
										<div>
											<div class="font-medium text-gray-900">{attachment.name}</div>
											<div class="text-xs text-gray-500">{attachment.type}</div>
										</div>
									</div>
									<a href={attachment.url} target="_blank" class="btn btn-ghost btn-sm">Öffnen</a>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-sm text-gray-500">Noch keine Dokumente hochgeladen.</p>
					{/if}
				</div>
			</div>

			<!-- Sidebar: Timeline -->
			<div class="space-y-6">
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
					<h2 class="text-lg font-semibold text-gray-900 mb-4">Verlauf</h2>

					{#if antrag.timeline && antrag.timeline.length > 0}
						<div class="relative">
							<div class="absolute left-2 top-0 bottom-0 w-0.5 bg-gray-200"></div>
							<div class="space-y-4">
								{#each antrag.timeline as entry, index}
									{@const entryStatus = statusLabels[entry.status]}
									<div class="relative flex gap-4">
										<div class="relative z-10 w-4 h-4 rounded-full {entryStatus?.color || 'bg-gray-200'} border-2 border-white"></div>
										<div class="flex-1 pb-4">
											<div class="text-sm font-medium text-gray-900">{entryStatus?.label || entry.status}</div>
											<div class="text-xs text-gray-500">{new Date(entry.date).toLocaleString('de-AT')}</div>
											{#if entry.description}
												<div class="text-sm text-gray-600 mt-1">{entry.description}</div>
											{/if}
										</div>
									</div>
								{/each}
							</div>
						</div>
					{:else}
						<p class="text-sm text-gray-500">Keine Einträge vorhanden.</p>
					{/if}
				</div>

				<!-- Quick Actions -->
				<div class="bg-blue-50 rounded-lg p-6">
					<h3 class="font-semibold text-gray-900 mb-3">Aktionen</h3>
					<div class="space-y-2">
						{#if antrag.status === 'drafting'}
							<button on:click={() => { newStatus = 'submitted'; updateStatus(); }} class="btn btn-primary w-full">
								<Send class="w-4 h-4 mr-2" />
								Antrag einreichen
							</button>
						{/if}
						<a href="/foerderungen/{antrag.foerderung_id}" class="btn btn-outline w-full">
							Zur Förderung
						</a>
					</div>
				</div>

				<!-- Info -->
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 text-sm text-gray-500">
					<div class="mb-2">
						<span class="text-gray-700">Erstellt:</span> {new Date(antrag.created_at).toLocaleString('de-AT')}
					</div>
					<div>
						<span class="text-gray-700">Aktualisiert:</span> {new Date(antrag.updated_at).toLocaleString('de-AT')}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
