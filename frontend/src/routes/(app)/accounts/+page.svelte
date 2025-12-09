<script lang="ts">
	import { formatDate, formatDateTime } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	// Account types
	type AccountType = 'finanzOnline' | 'elda' | 'firmenbuch';
	type AccountStatus = 'active' | 'syncing' | 'error' | 'pending';

	interface Account {
		id: string;
		name: string;
		type: AccountType;
		teilnehmerId: string;
		status: AccountStatus;
		lastSync: Date | null;
		documentsCount: number;
		tags: string[];
	}

	// Mock data
	let accounts = $state<Account[]>([
		{
			id: '1',
			name: 'Muster GmbH',
			type: 'finanzOnline',
			teilnehmerId: '123456789',
			status: 'active',
			lastSync: new Date(Date.now() - 1800000),
			documentsCount: 45,
			tags: ['Mandant', 'Wien']
		},
		{
			id: '2',
			name: 'Test AG',
			type: 'finanzOnline',
			teilnehmerId: '987654321',
			status: 'syncing',
			lastSync: new Date(Date.now() - 3600000),
			documentsCount: 23,
			tags: ['Mandant']
		},
		{
			id: '3',
			name: 'Demo GmbH',
			type: 'finanzOnline',
			teilnehmerId: '456789123',
			status: 'error',
			lastSync: new Date(Date.now() - 86400000),
			documentsCount: 12,
			tags: ['Test']
		},
		{
			id: '4',
			name: 'Beispiel KG',
			type: 'elda',
			teilnehmerId: 'E-123456',
			status: 'active',
			lastSync: new Date(Date.now() - 7200000),
			documentsCount: 8,
			tags: []
		}
	]);

	let searchQuery = $state('');
	let selectedType = $state<AccountType | 'all'>('all');
	let selectedStatus = $state<AccountStatus | 'all'>('all');

	let filteredAccounts = $derived(
		accounts.filter(account => {
			const matchesSearch = !searchQuery ||
				account.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				account.teilnehmerId.includes(searchQuery);
			const matchesType = selectedType === 'all' || account.type === selectedType;
			const matchesStatus = selectedStatus === 'all' || account.status === selectedStatus;
			return matchesSearch && matchesType && matchesStatus;
		})
	);

	function getStatusVariant(status: AccountStatus): 'success' | 'warning' | 'error' | 'info' {
		switch (status) {
			case 'active': return 'success';
			case 'syncing': return 'info';
			case 'error': return 'error';
			case 'pending': return 'warning';
		}
	}

	function getStatusLabel(status: AccountStatus): string {
		switch (status) {
			case 'active': return 'Active';
			case 'syncing': return 'Syncing...';
			case 'error': return 'Error';
			case 'pending': return 'Pending';
		}
	}

	function getTypeLabel(type: AccountType): string {
		switch (type) {
			case 'finanzOnline': return 'FinanzOnline';
			case 'elda': return 'ELDA';
			case 'firmenbuch': return 'Firmenbuch';
		}
	}

	function getTypeIcon(type: AccountType): string {
		switch (type) {
			case 'finanzOnline':
				return '<rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/>';
			case 'elda':
				return '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>';
			case 'firmenbuch':
				return '<rect width="16" height="20" x="4" y="2" rx="2" ry="2"/><path d="M9 22v-4h6v4"/><path d="M8 6h.01"/><path d="M16 6h.01"/><path d="M12 6h.01"/><path d="M12 10h.01"/><path d="M12 14h.01"/><path d="M16 10h.01"/><path d="M16 14h.01"/><path d="M8 10h.01"/><path d="M8 14h.01"/>';
		}
	}

	async function syncAccount(accountId: string) {
		// Update status to syncing
		accounts = accounts.map(a =>
			a.id === accountId ? { ...a, status: 'syncing' as AccountStatus } : a
		);

		// Simulate sync
		await new Promise(resolve => setTimeout(resolve, 2000));

		// Update status back to active
		accounts = accounts.map(a =>
			a.id === accountId ? { ...a, status: 'active' as AccountStatus, lastSync: new Date() } : a
		);
	}
</script>

<div class="max-w-7xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<p class="text-[var(--color-ink-muted)]">
				{accounts.length} accounts configured
			</p>
		</div>
		<Button href="/accounts/new">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 5v14M5 12h14"/>
			</svg>
			Add Account
		</Button>
	</div>

	<!-- Filters -->
	<Card>
		<div class="flex flex-col sm:flex-row gap-4">
			<div class="flex-1">
				<Input
					type="search"
					bind:value={searchQuery}
					placeholder="Search accounts..."
				/>
			</div>
			<div class="flex gap-3">
				<select
					bind:value={selectedType}
					class="input h-10 w-40"
				>
					<option value="all">All Types</option>
					<option value="finanzOnline">FinanzOnline</option>
					<option value="elda">ELDA</option>
					<option value="firmenbuch">Firmenbuch</option>
				</select>
				<select
					bind:value={selectedStatus}
					class="input h-10 w-40"
				>
					<option value="all">All Status</option>
					<option value="active">Active</option>
					<option value="syncing">Syncing</option>
					<option value="error">Error</option>
					<option value="pending">Pending</option>
				</select>
			</div>
		</div>
	</Card>

	<!-- Accounts list -->
	{#if filteredAccounts.length === 0}
		<Card class="text-center py-12">
			<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<rect width="16" height="20" x="4" y="2" rx="2" ry="2"/>
				<path d="M9 22v-4h6v4"/>
			</svg>
			<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">No accounts found</h3>
			<p class="mt-2 text-[var(--color-ink-muted)]">
				{searchQuery || selectedType !== 'all' || selectedStatus !== 'all'
					? 'Try adjusting your search or filters.'
					: 'Get started by adding your first account.'}
			</p>
			{#if !searchQuery && selectedType === 'all' && selectedStatus === 'all'}
				<Button href="/accounts/new" class="mt-4">
					Add Account
				</Button>
			{/if}
		</Card>
	{:else}
		<div class="grid gap-4">
			{#each filteredAccounts as account, i}
				<Card hover class="stagger-{Math.min(i + 1, 5)}">
					<div class="flex items-center gap-6">
						<!-- Icon -->
						<div class="w-12 h-12 rounded-xl bg-[var(--color-paper-inset)] flex items-center justify-center flex-shrink-0">
							<svg class="w-6 h-6 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
								{@html getTypeIcon(account.type)}
							</svg>
						</div>

						<!-- Info -->
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-3">
								<a href="/accounts/{account.id}" class="text-lg font-medium text-[var(--color-ink)] hover:text-[var(--color-accent)] transition-colors">
									{account.name}
								</a>
								<Badge variant={getStatusVariant(account.status)} dot>
									{getStatusLabel(account.status)}
								</Badge>
							</div>
							<div class="flex items-center gap-4 mt-1 text-sm text-[var(--color-ink-muted)]">
								<span>{getTypeLabel(account.type)}</span>
								<span>•</span>
								<span>ID: {account.teilnehmerId}</span>
								{#if account.lastSync}
									<span>•</span>
									<span>Last sync: {formatDateTime(account.lastSync)}</span>
								{/if}
							</div>
							{#if account.tags.length > 0}
								<div class="flex gap-1.5 mt-2">
									{#each account.tags as tag}
										<Badge size="sm">{tag}</Badge>
									{/each}
								</div>
							{/if}
						</div>

						<!-- Stats -->
						<div class="hidden sm:flex items-center gap-8 text-center">
							<div>
								<p class="text-2xl font-semibold text-[var(--color-ink)]">{account.documentsCount}</p>
								<p class="text-xs text-[var(--color-ink-muted)]">Documents</p>
							</div>
						</div>

						<!-- Actions -->
						<div class="flex items-center gap-2">
							<Button
								variant="ghost"
								size="sm"
								disabled={account.status === 'syncing'}
								onclick={() => syncAccount(account.id)}
							>
								<svg class="w-4 h-4" class:animate-spin={account.status === 'syncing'} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
									<path d="M3 3v5h5"/>
									<path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/>
									<path d="M16 16h5v5"/>
								</svg>
							</Button>
							<Button variant="ghost" size="sm" href="/accounts/{account.id}">
								<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M5 12h14M12 5l7 7-7 7"/>
								</svg>
							</Button>
						</div>
					</div>
				</Card>
			{/each}
		</div>
	{/if}
</div>
