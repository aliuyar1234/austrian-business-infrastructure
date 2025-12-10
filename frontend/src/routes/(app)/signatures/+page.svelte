<script lang="ts">
	import { formatDate } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	type RequestStatus = 'pending' | 'in_progress' | 'completed' | 'expired' | 'cancelled';

	interface Signer {
		id: string;
		name: string;
		email: string;
		status: 'pending' | 'notified' | 'signed' | 'expired';
		signedAt?: Date;
	}

	interface SignatureRequest {
		id: string;
		title: string;
		documentName: string;
		status: RequestStatus;
		signers: Signer[];
		createdAt: Date;
		expiresAt: Date;
		completedAt?: Date;
	}

	// Mock data
	let requests = $state<SignatureRequest[]>([
		{
			id: '1',
			title: 'Arbeitsvertrag Max Mustermann',
			documentName: 'arbeitsvertrag_mustermann.pdf',
			status: 'pending',
			signers: [
				{ id: '1', name: 'Max Mustermann', email: 'max@example.com', status: 'notified' },
				{ id: '2', name: 'Geschaeftsfuehrer', email: 'gf@company.at', status: 'pending' }
			],
			createdAt: new Date(),
			expiresAt: new Date(Date.now() + 14 * 86400000)
		},
		{
			id: '2',
			title: 'Mietvertrag Buero Wien',
			documentName: 'mietvertrag_wien.pdf',
			status: 'in_progress',
			signers: [
				{ id: '3', name: 'Anna Schmidt', email: 'anna@example.com', status: 'signed', signedAt: new Date(Date.now() - 86400000) },
				{ id: '4', name: 'Peter Mueller', email: 'peter@example.com', status: 'notified' }
			],
			createdAt: new Date(Date.now() - 172800000),
			expiresAt: new Date(Date.now() + 12 * 86400000)
		},
		{
			id: '3',
			title: 'Jahresabschluss 2024',
			documentName: 'jahresabschluss_2024.pdf',
			status: 'completed',
			signers: [
				{ id: '5', name: 'Steuerberater', email: 'stb@steuerberatung.at', status: 'signed', signedAt: new Date(Date.now() - 259200000) }
			],
			createdAt: new Date(Date.now() - 604800000),
			expiresAt: new Date(Date.now() + 7 * 86400000),
			completedAt: new Date(Date.now() - 259200000)
		},
		{
			id: '4',
			title: 'Kundenvertrag ABC GmbH',
			documentName: 'vertrag_abc.pdf',
			status: 'expired',
			signers: [
				{ id: '6', name: 'ABC Kunde', email: 'kunde@abc.at', status: 'expired' }
			],
			createdAt: new Date(Date.now() - 1209600000),
			expiresAt: new Date(Date.now() - 86400000)
		}
	]);

	let searchQuery = $state('');
	let selectedStatus = $state<RequestStatus | 'all'>('all');

	let filteredRequests = $derived(
		requests.filter(req => {
			const matchesSearch = !searchQuery ||
				req.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
				req.documentName.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesStatus = selectedStatus === 'all' || req.status === selectedStatus;
			return matchesSearch && matchesStatus;
		})
	);

	let pendingCount = $derived(requests.filter(r => r.status === 'pending' || r.status === 'in_progress').length);

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

	function getSignerStatusIcon(status: Signer['status']): string {
		switch (status) {
			case 'signed':
				return '<polyline points="20 6 9 17 4 12"/>';
			case 'notified':
				return '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>';
			case 'expired':
				return '<circle cx="12" cy="12" r="10"/><line x1="15" x2="9" y1="9" y2="15"/><line x1="9" x2="15" y1="9" y2="15"/>';
			default:
				return '<circle cx="12" cy="12" r="10"/>';
		}
	}

	function getSignerProgress(signers: Signer[]): string {
		const signed = signers.filter(s => s.status === 'signed').length;
		return `${signed}/${signers.length}`;
	}

	async function sendReminder(requestId: string) {
		// TODO: Call API to send reminder
		alert(`Erinnerung gesendet fuer Anfrage ${requestId}`);
	}
</script>

<div class="max-w-7xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold text-[var(--color-ink)]">Digitale Signaturen</h1>
			<p class="text-[var(--color-ink-muted)]">
				{requests.length} Signaturanfragen
				{#if pendingCount > 0}
					<span class="ml-2 text-[var(--color-accent)] font-medium">({pendingCount} offen)</span>
				{/if}
			</p>
		</div>
		<div class="flex gap-2">
			<Button variant="secondary" href="/signatures/verify">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M9 12l2 2 4-4"/>
					<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
				</svg>
				Signatur pruefen
			</Button>
			<Button href="/signatures/new">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="12" x2="12" y1="5" y2="19"/>
					<line x1="5" x2="19" y1="12" y2="12"/>
				</svg>
				Neue Signaturanfrage
			</Button>
		</div>
	</div>

	<!-- Filters -->
	<Card>
		<div class="flex flex-col sm:flex-row gap-4">
			<div class="flex-1">
				<Input
					type="search"
					bind:value={searchQuery}
					placeholder="Dokumente suchen..."
				/>
			</div>
			<div class="flex flex-wrap gap-3">
				<select bind:value={selectedStatus} class="input h-10 w-40">
					<option value="all">Alle Status</option>
					<option value="pending">Ausstehend</option>
					<option value="in_progress">In Bearbeitung</option>
					<option value="completed">Abgeschlossen</option>
					<option value="expired">Abgelaufen</option>
					<option value="cancelled">Storniert</option>
				</select>
			</div>
		</div>
	</Card>

	<!-- Requests list -->
	{#if filteredRequests.length === 0}
		<Card class="text-center py-12">
			<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
			</svg>
			<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">Keine Signaturanfragen</h3>
			<p class="mt-2 text-[var(--color-ink-muted)]">
				Erstellen Sie eine neue Signaturanfrage oder passen Sie Ihre Filter an.
			</p>
			<Button class="mt-4" href="/signatures/new">
				Neue Signaturanfrage
			</Button>
		</Card>
	{:else}
		<Card padding="none">
			<table class="table">
				<thead>
					<tr>
						<th>Dokument</th>
						<th>Unterzeichner</th>
						<th>Status</th>
						<th>Erstellt</th>
						<th>Ablauf</th>
						<th class="w-28"></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredRequests as req}
						<tr>
							<td>
								<div class="flex items-center gap-3">
									<div class="w-9 h-9 rounded-lg bg-[var(--color-paper-inset)] flex items-center justify-center flex-shrink-0">
										<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
											<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
										</svg>
									</div>
									<div class="min-w-0">
										<a
											href="/signatures/{req.id}"
											class="text-sm font-medium text-[var(--color-ink)] hover:text-[var(--color-accent)] transition-colors truncate block"
										>
											{req.title}
										</a>
										<span class="text-xs text-[var(--color-ink-muted)]">{req.documentName}</span>
									</div>
								</div>
							</td>
							<td>
								<div class="flex items-center gap-2">
									<div class="flex -space-x-2">
										{#each req.signers.slice(0, 3) as signer}
											<div
												class="w-7 h-7 rounded-full bg-[var(--color-paper-inset)] border-2 border-[var(--color-paper)] flex items-center justify-center"
												title="{signer.name} ({signer.status})"
											>
												<svg class="w-3.5 h-3.5 {signer.status === 'signed' ? 'text-[var(--color-success)]' : signer.status === 'expired' ? 'text-[var(--color-error)]' : 'text-[var(--color-ink-muted)]'}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
													{@html getSignerStatusIcon(signer.status)}
												</svg>
											</div>
										{/each}
									</div>
									<span class="text-sm text-[var(--color-ink-muted)]">{getSignerProgress(req.signers)}</span>
								</div>
							</td>
							<td>
								<Badge variant={getStatusVariant(req.status)} size="sm">
									{getStatusLabel(req.status)}
								</Badge>
							</td>
							<td>
								<span class="text-sm text-[var(--color-ink-muted)]">{formatDate(req.createdAt)}</span>
							</td>
							<td>
								{#if req.status === 'completed' && req.completedAt}
									<span class="text-sm text-[var(--color-success)]">Abgeschlossen</span>
								{:else if req.status === 'expired'}
									<span class="text-sm text-[var(--color-error)]">Abgelaufen</span>
								{:else}
									{@const daysLeft = Math.ceil((req.expiresAt.getTime() - Date.now()) / 86400000)}
									<Badge variant={daysLeft <= 3 ? 'error' : daysLeft <= 7 ? 'warning' : 'default'} size="sm">
										{daysLeft} Tage
									</Badge>
								{/if}
							</td>
							<td>
								<div class="flex items-center justify-end gap-1">
									{#if req.status === 'pending' || req.status === 'in_progress'}
										<Button variant="ghost" size="sm" onclick={() => sendReminder(req.id)} title="Erinnerung senden">
											<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
												<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/>
												<path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>
											</svg>
										</Button>
									{/if}
									<Button variant="ghost" size="sm" href="/signatures/{req.id}">
										<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/>
											<circle cx="12" cy="12" r="3"/>
										</svg>
									</Button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</Card>
	{/if}
</div>
