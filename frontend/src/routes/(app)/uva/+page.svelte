<script lang="ts">
	import { formatDate, formatCurrency } from '$lib/utils';
	import { getStatusLabel, getStatusVariant, type UVAStatus } from '$lib/utils/status';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';

	interface UVA {
		id: string;
		accountId: string;
		accountName: string;
		period: string; // e.g., "01/2025"
		year: number;
		month: number;
		status: UVAStatus;
		submittedAt?: Date;
		foReference?: string;
		kennzahlen: {
			kz000: number; // Umsätze
			kz022: number; // Erwerbsteuer
			kz060: number; // Vorsteuer
			kz083: number; // Zahllast/Gutschrift
		};
		createdAt: Date;
		updatedAt: Date;
	}

	// Mock data
	let uvas = $state<UVA[]>([
		{
			id: '1',
			accountId: '1',
			accountName: 'Muster GmbH',
			period: '12/2024',
			year: 2024,
			month: 12,
			status: 'submitted',
			submittedAt: new Date(Date.now() - 2 * 86400000),
			foReference: 'FO-2024-1234567',
			kennzahlen: { kz000: 125000, kz022: 0, kz060: 8750, kz083: 16250 },
			createdAt: new Date(Date.now() - 5 * 86400000),
			updatedAt: new Date(Date.now() - 2 * 86400000),
		},
		{
			id: '2',
			accountId: '1',
			accountName: 'Muster GmbH',
			period: '01/2025',
			year: 2025,
			month: 1,
			status: 'draft',
			kennzahlen: { kz000: 0, kz022: 0, kz060: 0, kz083: 0 },
			createdAt: new Date(),
			updatedAt: new Date(),
		},
		{
			id: '3',
			accountId: '2',
			accountName: 'Test AG',
			period: '12/2024',
			year: 2024,
			month: 12,
			status: 'accepted',
			submittedAt: new Date(Date.now() - 10 * 86400000),
			foReference: 'FO-2024-7654321',
			kennzahlen: { kz000: 85000, kz022: 1500, kz060: 12500, kz083: 6000 },
			createdAt: new Date(Date.now() - 15 * 86400000),
			updatedAt: new Date(Date.now() - 10 * 86400000),
		},
	]);

	let selectedAccount = $state<string>('all');
	let selectedStatus = $state<UVAStatus | 'all'>('all');

	let filteredUVAs = $derived(
		uvas.filter(u => {
			if (selectedAccount !== 'all' && u.accountId !== selectedAccount) return false;
			if (selectedStatus !== 'all' && u.status !== selectedStatus) return false;
			return true;
		})
	);

	let accounts = $derived([...new Set(uvas.map(u => ({ id: u.accountId, name: u.accountName })))]);
</script>

<svelte:head>
	<title>UVA - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-7xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<h1 class="text-xl font-semibold text-[var(--color-ink)]">Umsatzsteuervoranmeldung</h1>
			<p class="text-sm text-[var(--color-ink-muted)]">
				Create and submit VAT returns to FinanzOnline
			</p>
		</div>
		<Button href="/uva/new">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 5v14M5 12h14"/>
			</svg>
			New UVA
		</Button>
	</div>

	<!-- Filters -->
	<Card>
		<div class="flex flex-wrap gap-4">
			<div>
				<label for="account-filter" class="label">Account</label>
				<select id="account-filter" bind:value={selectedAccount} class="input h-10 w-48">
					<option value="all">All accounts</option>
					{#each accounts as acc}
						<option value={acc.id}>{acc.name}</option>
					{/each}
				</select>
			</div>
			<div>
				<label for="status-filter" class="label">Status</label>
				<select id="status-filter" bind:value={selectedStatus} class="input h-10 w-36">
					<option value="all">All status</option>
					<option value="draft">Draft</option>
					<option value="submitted">Submitted</option>
					<option value="accepted">Accepted</option>
					<option value="rejected">Rejected</option>
				</select>
			</div>
		</div>
	</Card>

	<!-- UVA list -->
	{#if filteredUVAs.length === 0}
		<EmptyState
			title="No UVA returns found"
			description="Create your first UVA return to get started."
			actionHref="/uva/new"
			actionLabel="Create UVA"
		>
			{#snippet icon()}
				<svg class="w-12 h-12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<rect width="18" height="18" x="3" y="3" rx="2"/>
					<path d="M7 7h10"/><path d="M7 12h10"/><path d="M7 17h10"/>
				</svg>
			{/snippet}
		</EmptyState>
	{:else}
		<div class="space-y-4">
			{#each filteredUVAs as uva}
				<Card hover onclick={() => {}}>
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-4">
							<div class="w-12 h-12 rounded-xl bg-[var(--color-paper-inset)] flex items-center justify-center">
								<span class="text-sm font-bold text-[var(--color-ink-muted)]">
									{uva.period.split('/')[0]}
								</span>
							</div>
							<div>
								<div class="flex items-center gap-2">
									<h3 class="font-medium text-[var(--color-ink)]">UVA {uva.period}</h3>
									<Badge variant={getStatusVariant(uva.status, 'uva')} size="sm">
										{getStatusLabel(uva.status, 'uva')}
									</Badge>
								</div>
								<p class="text-sm text-[var(--color-ink-muted)]">
									{uva.accountName}
									{#if uva.foReference}
										• {uva.foReference}
									{/if}
								</p>
							</div>
						</div>

						<div class="hidden sm:flex items-center gap-8">
							<div class="text-right">
								<p class="text-xs text-[var(--color-ink-muted)]">Umsätze</p>
								<p class="text-sm font-medium text-[var(--color-ink)]">{formatCurrency(uva.kennzahlen.kz000)}</p>
							</div>
							<div class="text-right">
								<p class="text-xs text-[var(--color-ink-muted)]">Zahllast</p>
								<p class={`text-sm font-medium ${uva.kennzahlen.kz083 >= 0 ? 'text-[var(--color-error)]' : 'text-[var(--color-success)]'}`}>
									{formatCurrency(uva.kennzahlen.kz083)}
								</p>
							</div>
							<div class="text-right min-w-[100px]">
								<p class="text-xs text-[var(--color-ink-muted)]">
									{uva.status === 'draft' ? 'Created' : 'Submitted'}
								</p>
								<p class="text-sm text-[var(--color-ink)]">
									{formatDate(uva.status === 'draft' ? uva.createdAt : uva.submittedAt!)}
								</p>
							</div>
						</div>

						<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="m9 18 6-6-6-6"/>
						</svg>
					</div>
				</Card>
			{/each}
		</div>
	{/if}
</div>
