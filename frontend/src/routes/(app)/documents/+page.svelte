<script lang="ts">
	import { formatDate } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	type DocumentType = 'bescheid' | 'ersuchen' | 'info' | 'quittung' | 'other';
	type DocumentStatus = 'unread' | 'read' | 'archived';

	interface Document {
		id: string;
		title: string;
		type: DocumentType;
		status: DocumentStatus;
		accountId: string;
		accountName: string;
		receivedAt: Date;
		hasDeadline: boolean;
		deadline?: Date;
	}

	// Mock data
	let documents = $state<Document[]>([
		{
			id: '1',
			title: 'Bescheid Umsatzsteuer 2024',
			type: 'bescheid',
			status: 'unread',
			accountId: '1',
			accountName: 'Muster GmbH',
			receivedAt: new Date(),
			hasDeadline: false
		},
		{
			id: '2',
			title: 'Ergänzungsersuchen zur Umsatzsteuererklärung',
			type: 'ersuchen',
			status: 'unread',
			accountId: '2',
			accountName: 'Test AG',
			receivedAt: new Date(Date.now() - 86400000),
			hasDeadline: true,
			deadline: new Date(Date.now() + 14 * 86400000)
		},
		{
			id: '3',
			title: 'Vorauszahlungsbescheid Q1 2025',
			type: 'bescheid',
			status: 'read',
			accountId: '1',
			accountName: 'Muster GmbH',
			receivedAt: new Date(Date.now() - 172800000),
			hasDeadline: false
		},
		{
			id: '4',
			title: 'Informationsschreiben Finanzamt',
			type: 'info',
			status: 'read',
			accountId: '3',
			accountName: 'Demo GmbH',
			receivedAt: new Date(Date.now() - 259200000),
			hasDeadline: false
		},
		{
			id: '5',
			title: 'Quittung UVA Dezember 2024',
			type: 'quittung',
			status: 'read',
			accountId: '1',
			accountName: 'Muster GmbH',
			receivedAt: new Date(Date.now() - 345600000),
			hasDeadline: false
		}
	]);

	let searchQuery = $state('');
	let selectedType = $state<DocumentType | 'all'>('all');
	let selectedStatus = $state<DocumentStatus | 'all'>('all');
	let selectedAccount = $state<string>('all');

	let filteredDocuments = $derived(
		documents.filter(doc => {
			const matchesSearch = !searchQuery ||
				doc.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
				doc.accountName.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesType = selectedType === 'all' || doc.type === selectedType;
			const matchesStatus = selectedStatus === 'all' || doc.status === selectedStatus;
			const matchesAccount = selectedAccount === 'all' || doc.accountId === selectedAccount;
			return matchesSearch && matchesType && matchesStatus && matchesAccount;
		})
	);

	let unreadCount = $derived(documents.filter(d => d.status === 'unread').length);

	function getTypeLabel(type: DocumentType): string {
		switch (type) {
			case 'bescheid': return 'Bescheid';
			case 'ersuchen': return 'Ersuchen';
			case 'info': return 'Information';
			case 'quittung': return 'Quittung';
			case 'other': return 'Sonstige';
		}
	}

	function getTypeVariant(type: DocumentType): 'default' | 'warning' | 'info' | 'success' {
		switch (type) {
			case 'bescheid': return 'default';
			case 'ersuchen': return 'warning';
			case 'info': return 'info';
			case 'quittung': return 'success';
			default: return 'default';
		}
	}

	function getDocumentIcon(type: DocumentType): string {
		switch (type) {
			case 'bescheid':
				return '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" x2="8" y1="13" y2="13"/><line x1="16" x2="8" y1="17" y2="17"/>';
			case 'ersuchen':
				return '<circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><path d="M12 17h.01"/>';
			case 'info':
				return '<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>';
			case 'quittung':
				return '<polyline points="20 6 9 17 4 12"/>';
			default:
				return '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>';
		}
	}

	async function markAsRead(docId: string) {
		documents = documents.map(d =>
			d.id === docId ? { ...d, status: 'read' as DocumentStatus } : d
		);
	}
</script>

<div class="max-w-7xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<p class="text-[var(--color-ink-muted)]">
				{documents.length} documents total
				{#if unreadCount > 0}
					<span class="ml-2 text-[var(--color-accent)] font-medium">({unreadCount} unread)</span>
				{/if}
			</p>
		</div>
		<div class="flex gap-2">
			<Button variant="secondary">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
					<polyline points="7 10 12 15 17 10"/>
					<line x1="12" x2="12" y1="15" y2="3"/>
				</svg>
				Export
			</Button>
			<Button variant="ghost">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
					<path d="M3 3v5h5"/>
				</svg>
				Sync All
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
					placeholder="Search documents..."
				/>
			</div>
			<div class="flex flex-wrap gap-3">
				<select bind:value={selectedType} class="input h-10 w-36">
					<option value="all">All Types</option>
					<option value="bescheid">Bescheid</option>
					<option value="ersuchen">Ersuchen</option>
					<option value="info">Information</option>
					<option value="quittung">Quittung</option>
				</select>
				<select bind:value={selectedStatus} class="input h-10 w-32">
					<option value="all">All Status</option>
					<option value="unread">Unread</option>
					<option value="read">Read</option>
					<option value="archived">Archived</option>
				</select>
			</div>
		</div>
	</Card>

	<!-- Documents list -->
	{#if filteredDocuments.length === 0}
		<Card class="text-center py-12">
			<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
				<polyline points="14 2 14 8 20 8"/>
			</svg>
			<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">No documents found</h3>
			<p class="mt-2 text-[var(--color-ink-muted)]">
				Try adjusting your search or filters.
			</p>
		</Card>
	{:else}
		<Card padding="none">
			<table class="table">
				<thead>
					<tr>
						<th>Document</th>
						<th>Account</th>
						<th>Type</th>
						<th>Received</th>
						<th>Deadline</th>
						<th class="w-20"></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredDocuments as doc}
						<tr class={doc.status === 'unread' ? 'bg-[var(--color-accent-muted)]/30' : ''}>
							<td>
								<div class="flex items-center gap-3">
									<div class="w-9 h-9 rounded-lg bg-[var(--color-paper-inset)] flex items-center justify-center flex-shrink-0">
										<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
											{@html getDocumentIcon(doc.type)}
										</svg>
									</div>
									<div class="min-w-0">
										<a
											href="/documents/{doc.id}"
											class="text-sm font-medium text-[var(--color-ink)] hover:text-[var(--color-accent)] transition-colors truncate block"
										>
											{doc.title}
										</a>
										{#if doc.status === 'unread'}
											<span class="text-xs text-[var(--color-accent)] font-medium">New</span>
										{/if}
									</div>
								</div>
							</td>
							<td>
								<span class="text-sm text-[var(--color-ink-secondary)]">{doc.accountName}</span>
							</td>
							<td>
								<Badge variant={getTypeVariant(doc.type)} size="sm">
									{getTypeLabel(doc.type)}
								</Badge>
							</td>
							<td>
								<span class="text-sm text-[var(--color-ink-muted)]">{formatDate(doc.receivedAt)}</span>
							</td>
							<td>
								{#if doc.hasDeadline && doc.deadline}
									{@const daysLeft = Math.ceil((doc.deadline.getTime() - Date.now()) / 86400000)}
									<Badge variant={daysLeft <= 7 ? 'error' : 'warning'} size="sm" dot>
										{formatDate(doc.deadline)}
									</Badge>
								{:else}
									<span class="text-sm text-[var(--color-ink-muted)]">—</span>
								{/if}
							</td>
							<td>
								<div class="flex items-center justify-end gap-1">
									<Button variant="ghost" size="sm" href="/documents/{doc.id}">
										<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/>
											<circle cx="12" cy="12" r="3"/>
										</svg>
									</Button>
									<Button variant="ghost" size="sm">
										<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
											<polyline points="7 10 12 15 17 10"/>
											<line x1="12" x2="12" y1="15" y2="3"/>
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
