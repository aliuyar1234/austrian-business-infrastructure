<script lang="ts">
	import { page } from '$app/stores';
	import { formatDate } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	type RequestStatus = 'pending' | 'in_progress' | 'completed' | 'expired' | 'cancelled';
	type SignerStatus = 'pending' | 'notified' | 'signed' | 'expired';

	interface Signer {
		id: string;
		name: string;
		email: string;
		status: SignerStatus;
		reason?: string;
		signedAt?: Date;
		notifiedAt?: Date;
		reminderCount: number;
		lastReminderAt?: Date;
	}

	interface SignatureRequest {
		id: string;
		title: string;
		documentName: string;
		documentId: string;
		status: RequestStatus;
		signers: Signer[];
		createdAt: Date;
		expiresAt: Date;
		completedAt?: Date;
		message?: string;
		createdByName: string;
	}

	// Mock data
	let request = $state<SignatureRequest>({
		id: $page.params.id,
		title: 'Arbeitsvertrag Max Mustermann',
		documentName: 'arbeitsvertrag_mustermann.pdf',
		documentId: 'doc-123',
		status: 'in_progress',
		signers: [
			{
				id: '1',
				name: 'Max Mustermann',
				email: 'max@example.com',
				status: 'signed',
				reason: 'Zustimmung',
				signedAt: new Date(Date.now() - 86400000),
				notifiedAt: new Date(Date.now() - 172800000),
				reminderCount: 0
			},
			{
				id: '2',
				name: 'Anna Schmidt',
				email: 'anna@example.com',
				status: 'notified',
				reason: 'Genehmigung',
				notifiedAt: new Date(Date.now() - 43200000),
				reminderCount: 1,
				lastReminderAt: new Date(Date.now() - 21600000)
			},
			{
				id: '3',
				name: 'Peter Mueller',
				email: 'peter@example.com',
				status: 'pending',
				reason: 'Kenntnisnahme',
				reminderCount: 0
			}
		],
		createdAt: new Date(Date.now() - 259200000),
		expiresAt: new Date(Date.now() + 11 * 86400000),
		message: 'Bitte unterschreiben Sie den beigefuegten Arbeitsvertrag.',
		createdByName: 'Admin User'
	});

	let isLoading = $state(false);
	let showCancelDialog = $state(false);

	function getStatusLabel(status: RequestStatus): string {
		switch (status) {
			case 'pending': return 'Ausstehend';
			case 'in_progress': return 'In Bearbeitung';
			case 'completed': return 'Abgeschlossen';
			case 'expired': return 'Abgelaufen';
			case 'cancelled': return 'Storniert';
		}
	}

	function getStatusVariant(status: RequestStatus): 'default' | 'warning' | 'info' | 'success' | 'error' {
		switch (status) {
			case 'pending': return 'warning';
			case 'in_progress': return 'info';
			case 'completed': return 'success';
			case 'expired': return 'error';
			case 'cancelled': return 'default';
		}
	}

	function getSignerStatusLabel(status: SignerStatus): string {
		switch (status) {
			case 'pending': return 'Wartet';
			case 'notified': return 'Eingeladen';
			case 'signed': return 'Signiert';
			case 'expired': return 'Abgelaufen';
		}
	}

	function getSignerStatusVariant(status: SignerStatus): 'default' | 'warning' | 'info' | 'success' | 'error' {
		switch (status) {
			case 'pending': return 'default';
			case 'notified': return 'info';
			case 'signed': return 'success';
			case 'expired': return 'error';
		}
	}

	let daysLeft = $derived(Math.ceil((request.expiresAt.getTime() - Date.now()) / 86400000));
	let signedCount = $derived(request.signers.filter(s => s.status === 'signed').length);
	let currentSigner = $derived(request.signers.find(s => s.status === 'notified'));

	async function sendReminder(signerId: string) {
		isLoading = true;
		try {
			// TODO: Call API to send reminder
			await new Promise(resolve => setTimeout(resolve, 1000));
			const signer = request.signers.find(s => s.id === signerId);
			if (signer) {
				signer.reminderCount++;
				signer.lastReminderAt = new Date();
			}
		} finally {
			isLoading = false;
		}
	}

	async function cancelRequest() {
		isLoading = true;
		try {
			// TODO: Call API to cancel request
			await new Promise(resolve => setTimeout(resolve, 1000));
			request.status = 'cancelled';
			showCancelDialog = false;
		} finally {
			isLoading = false;
		}
	}

	async function downloadDocument() {
		// TODO: Download the signed document
		alert('Download gestartet...');
	}
</script>

<div class="max-w-4xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex items-start justify-between gap-4">
		<div class="flex items-center gap-4">
			<Button variant="ghost" href="/signatures">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="m15 18-6-6 6-6"/>
				</svg>
			</Button>
			<div>
				<div class="flex items-center gap-3">
					<h1 class="text-2xl font-semibold text-[var(--color-ink)]">{request.title}</h1>
					<Badge variant={getStatusVariant(request.status)}>
						{getStatusLabel(request.status)}
					</Badge>
				</div>
				<p class="text-[var(--color-ink-muted)]">{request.documentName}</p>
			</div>
		</div>
		<div class="flex gap-2">
			{#if request.status === 'completed'}
				<Button onclick={downloadDocument}>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
						<polyline points="7 10 12 15 17 10"/>
						<line x1="12" x2="12" y1="15" y2="3"/>
					</svg>
					Signiertes Dokument
				</Button>
			{:else if request.status !== 'cancelled' && request.status !== 'expired'}
				<Button variant="secondary" onclick={() => showCancelDialog = true}>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10"/>
						<line x1="15" x2="9" y1="9" y2="15"/>
						<line x1="9" x2="15" y1="9" y2="15"/>
					</svg>
					Stornieren
				</Button>
			{/if}
		</div>
	</div>

	<!-- Status overview -->
	<div class="grid gap-4 sm:grid-cols-3">
		<Card class="text-center">
			<div class="text-3xl font-bold text-[var(--color-ink)]">{signedCount}/{request.signers.length}</div>
			<div class="text-sm text-[var(--color-ink-muted)]">Signaturen</div>
		</Card>
		<Card class="text-center">
			<div class="text-3xl font-bold {daysLeft <= 3 ? 'text-[var(--color-error)]' : daysLeft <= 7 ? 'text-[var(--color-warning)]' : 'text-[var(--color-ink)]'}">
				{#if request.status === 'completed'}
					<svg class="w-8 h-8 mx-auto text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<polyline points="20 6 9 17 4 12"/>
					</svg>
				{:else if request.status === 'expired' || request.status === 'cancelled'}
					&mdash;
				{:else}
					{daysLeft}
				{/if}
			</div>
			<div class="text-sm text-[var(--color-ink-muted)]">
				{#if request.status === 'completed'}
					Abgeschlossen
				{:else if request.status === 'expired'}
					Abgelaufen
				{:else if request.status === 'cancelled'}
					Storniert
				{:else}
					Tage verbleibend
				{/if}
			</div>
		</Card>
		<Card class="text-center">
			<div class="text-sm font-medium text-[var(--color-ink)]">
				{#if currentSigner}
					{currentSigner.name}
				{:else if request.status === 'completed'}
					Alle signiert
				{:else}
					&mdash;
				{/if}
			</div>
			<div class="text-sm text-[var(--color-ink-muted)]">Aktueller Unterzeichner</div>
		</Card>
	</div>

	<!-- Signers timeline -->
	<Card>
		<h2 class="text-lg font-medium text-[var(--color-ink)] mb-4">Unterzeichner</h2>

		<div class="space-y-4">
			{#each request.signers as signer, index}
				<div class="relative pl-8 pb-4 {index < request.signers.length - 1 ? 'border-l-2 border-[var(--color-border)]' : ''} ml-3">
					<!-- Timeline dot -->
					<div class="absolute -left-3 top-0 w-6 h-6 rounded-full flex items-center justify-center {
						signer.status === 'signed' ? 'bg-[var(--color-success)] text-white' :
						signer.status === 'notified' ? 'bg-[var(--color-accent)] text-white' :
						signer.status === 'expired' ? 'bg-[var(--color-error)] text-white' :
						'bg-[var(--color-paper-inset)] text-[var(--color-ink-muted)]'
					}">
						{#if signer.status === 'signed'}
							<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
								<polyline points="20 6 9 17 4 12"/>
							</svg>
						{:else if signer.status === 'notified'}
							<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
								<path d="M12 6v6l4 2"/>
							</svg>
						{:else if signer.status === 'expired'}
							<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
								<line x1="18" x2="6" y1="6" y2="18"/>
								<line x1="6" x2="18" y1="6" y2="18"/>
							</svg>
						{:else}
							<span class="text-xs font-medium">{index + 1}</span>
						{/if}
					</div>

					<div class="flex items-start justify-between gap-4">
						<div>
							<div class="flex items-center gap-2">
								<span class="font-medium text-[var(--color-ink)]">{signer.name}</span>
								<Badge variant={getSignerStatusVariant(signer.status)} size="sm">
									{getSignerStatusLabel(signer.status)}
								</Badge>
							</div>
							<div class="text-sm text-[var(--color-ink-muted)]">{signer.email}</div>
							{#if signer.reason}
								<div class="text-sm text-[var(--color-ink-secondary)] mt-1">Grund: {signer.reason}</div>
							{/if}
							{#if signer.signedAt}
								<div class="text-xs text-[var(--color-ink-muted)] mt-2">
									Signiert am {formatDate(signer.signedAt)}
								</div>
							{:else if signer.notifiedAt}
								<div class="text-xs text-[var(--color-ink-muted)] mt-2">
									Eingeladen am {formatDate(signer.notifiedAt)}
									{#if signer.reminderCount > 0}
										<span class="ml-2">({signer.reminderCount} Erinnerung{signer.reminderCount > 1 ? 'en' : ''} gesendet)</span>
									{/if}
								</div>
							{/if}
						</div>
						{#if signer.status === 'notified' && request.status !== 'cancelled' && request.status !== 'expired'}
							<Button
								variant="secondary"
								size="sm"
								onclick={() => sendReminder(signer.id)}
								disabled={isLoading}
							>
								<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/>
									<path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>
								</svg>
								Erinnerung
							</Button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	</Card>

	<!-- Request details -->
	<Card>
		<h2 class="text-lg font-medium text-[var(--color-ink)] mb-4">Details</h2>

		<dl class="grid gap-4 sm:grid-cols-2">
			<div>
				<dt class="text-sm text-[var(--color-ink-muted)]">Erstellt von</dt>
				<dd class="text-sm font-medium text-[var(--color-ink)]">{request.createdByName}</dd>
			</div>
			<div>
				<dt class="text-sm text-[var(--color-ink-muted)]">Erstellt am</dt>
				<dd class="text-sm font-medium text-[var(--color-ink)]">{formatDate(request.createdAt)}</dd>
			</div>
			<div>
				<dt class="text-sm text-[var(--color-ink-muted)]">Gueltig bis</dt>
				<dd class="text-sm font-medium text-[var(--color-ink)]">{formatDate(request.expiresAt)}</dd>
			</div>
			{#if request.completedAt}
				<div>
					<dt class="text-sm text-[var(--color-ink-muted)]">Abgeschlossen am</dt>
					<dd class="text-sm font-medium text-[var(--color-ink)]">{formatDate(request.completedAt)}</dd>
				</div>
			{/if}
		</dl>

		{#if request.message}
			<div class="mt-4 pt-4 border-t border-[var(--color-border)]">
				<dt class="text-sm text-[var(--color-ink-muted)] mb-1">Nachricht an Unterzeichner</dt>
				<dd class="text-sm text-[var(--color-ink)]">{request.message}</dd>
			</div>
		{/if}
	</Card>

	<!-- Audit log -->
	<Card>
		<h2 class="text-lg font-medium text-[var(--color-ink)] mb-4">Aktivitaeten</h2>

		<div class="space-y-3">
			{#if request.status === 'completed' && request.completedAt}
				<div class="flex items-start gap-3 text-sm">
					<div class="w-8 h-8 rounded-full bg-[var(--color-success-muted)] flex items-center justify-center flex-shrink-0">
						<svg class="w-4 h-4 text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="20 6 9 17 4 12"/>
						</svg>
					</div>
					<div>
						<div class="text-[var(--color-ink)]">Anfrage abgeschlossen</div>
						<div class="text-[var(--color-ink-muted)]">{formatDate(request.completedAt)}</div>
					</div>
				</div>
			{/if}
			{#each [...request.signers].reverse() as signer}
				{#if signer.signedAt}
					<div class="flex items-start gap-3 text-sm">
						<div class="w-8 h-8 rounded-full bg-[var(--color-success-muted)] flex items-center justify-center flex-shrink-0">
							<svg class="w-4 h-4 text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
							</svg>
						</div>
						<div>
							<div class="text-[var(--color-ink)]">{signer.name} hat signiert</div>
							<div class="text-[var(--color-ink-muted)]">{formatDate(signer.signedAt)}</div>
						</div>
					</div>
				{/if}
				{#if signer.notifiedAt}
					<div class="flex items-start gap-3 text-sm">
						<div class="w-8 h-8 rounded-full bg-[var(--color-accent-muted)] flex items-center justify-center flex-shrink-0">
							<svg class="w-4 h-4 text-[var(--color-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
								<polyline points="22,6 12,13 2,6"/>
							</svg>
						</div>
						<div>
							<div class="text-[var(--color-ink)]">Einladung an {signer.name} gesendet</div>
							<div class="text-[var(--color-ink-muted)]">{formatDate(signer.notifiedAt)}</div>
						</div>
					</div>
				{/if}
			{/each}
			<div class="flex items-start gap-3 text-sm">
				<div class="w-8 h-8 rounded-full bg-[var(--color-paper-inset)] flex items-center justify-center flex-shrink-0">
					<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="12" x2="12" y1="5" y2="19"/>
						<line x1="5" x2="19" y1="12" y2="12"/>
					</svg>
				</div>
				<div>
					<div class="text-[var(--color-ink)]">Signaturanfrage erstellt von {request.createdByName}</div>
					<div class="text-[var(--color-ink-muted)]">{formatDate(request.createdAt)}</div>
				</div>
			</div>
		</div>
	</Card>
</div>

<!-- Cancel dialog -->
{#if showCancelDialog}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => showCancelDialog = false}>
		<Card class="max-w-md mx-4" onclick={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-medium text-[var(--color-ink)]">Signaturanfrage stornieren?</h3>
			<p class="mt-2 text-sm text-[var(--color-ink-muted)]">
				Diese Aktion kann nicht rueckgaengig gemacht werden. Alle offenen Einladungen werden ungueltig.
			</p>
			<div class="mt-4 flex justify-end gap-3">
				<Button variant="secondary" onclick={() => showCancelDialog = false}>Abbrechen</Button>
				<Button variant="primary" onclick={cancelRequest} disabled={isLoading}>
					{#if isLoading}
						Wird storniert...
					{:else}
						Stornieren
					{/if}
				</Button>
			</div>
		</Card>
	</div>
{/if}
