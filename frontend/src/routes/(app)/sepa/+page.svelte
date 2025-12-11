<script lang="ts">
	import { formatDate, formatCurrency } from '$lib/utils';
	import { getStatusLabel, getStatusVariant, type PaymentStatus } from '$lib/utils/status';
	import { toast } from '$lib/stores/toast';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';

	type PaymentType = 'credit' | 'debit';

	interface Payment {
		id: string;
		type: PaymentType;
		reference: string;
		creditor: string;
		amount: number;
		iban: string;
		status: PaymentStatus;
		executionDate: Date;
		createdAt: Date;
	}

	// Mock data
	let payments = $state<Payment[]>([
		{
			id: '1',
			type: 'credit',
			reference: 'SEPA-2024-001',
			creditor: 'Lieferant GmbH',
			amount: 5250.00,
			iban: 'AT61 1904 3002 3457 3201',
			status: 'executed',
			executionDate: new Date(Date.now() - 5 * 86400000),
			createdAt: new Date(Date.now() - 7 * 86400000),
		},
		{
			id: '2',
			type: 'credit',
			reference: 'SEPA-2024-002',
			creditor: 'Dienstleister AG',
			amount: 1800.00,
			iban: 'AT48 3200 0000 0123 4568',
			status: 'generated',
			executionDate: new Date(Date.now() + 2 * 86400000),
			createdAt: new Date(Date.now() - 1 * 86400000),
		},
		{
			id: '3',
			type: 'debit',
			reference: 'SEPA-DD-001',
			creditor: 'Kunde Muster',
			amount: 299.00,
			iban: 'AT89 1200 0100 1234 5678',
			status: 'draft',
			executionDate: new Date(Date.now() + 7 * 86400000),
			createdAt: new Date(),
		},
	]);

	let showNewPaymentModal = $state(false);
	let searchQuery = $state('');
	let selectedType = $state<PaymentType | 'all'>('all');
	let selectedStatus = $state<PaymentStatus | 'all'>('all');

	// New payment form
	let newPaymentType = $state<PaymentType>('credit');
	let creditorName = $state('');
	let creditorIban = $state('');
	let creditorBic = $state('');
	let amount = $state('');
	let reference = $state('');
	let executionDate = $state(new Date(Date.now() + 2 * 86400000).toISOString().split('T')[0]);
	let isGenerating = $state(false);

	let filteredPayments = $derived(
		payments.filter(p => {
			const matchesSearch = !searchQuery ||
				p.reference.toLowerCase().includes(searchQuery.toLowerCase()) ||
				p.creditor.toLowerCase().includes(searchQuery.toLowerCase()) ||
				p.iban.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesType = selectedType === 'all' || p.type === selectedType;
			const matchesStatus = selectedStatus === 'all' || p.status === selectedStatus;
			return matchesSearch && matchesType && matchesStatus;
		})
	);

	function formatIBAN(iban: string): string {
		return iban.replace(/(.{4})/g, '$1 ').trim();
	}

	function validateIBAN(iban: string): boolean {
		const cleaned = iban.replace(/\s/g, '').toUpperCase();
		return /^[A-Z]{2}\d{2}[A-Z0-9]{4,30}$/.test(cleaned);
	}

	async function handleGeneratePayment() {
		if (!creditorName.trim() || !creditorIban.trim() || !amount) {
			toast.error('Validation error', 'Please fill in all required fields');
			return;
		}

		if (!validateIBAN(creditorIban)) {
			toast.error('Invalid IBAN', 'Please enter a valid IBAN');
			return;
		}

		isGenerating = true;
		await new Promise(r => setTimeout(r, 1500));

		const newPayment: Payment = {
			id: String(Date.now()),
			type: newPaymentType,
			reference: reference || `SEPA-${new Date().getFullYear()}-${String(payments.length + 1).padStart(3, '0')}`,
			creditor: creditorName,
			amount: parseFloat(amount),
			iban: creditorIban.replace(/\s/g, '').toUpperCase(),
			status: 'generated',
			executionDate: new Date(executionDate),
			createdAt: new Date(),
		};

		payments = [newPayment, ...payments];
		isGenerating = false;
		showNewPaymentModal = false;

		// Reset form
		creditorName = '';
		creditorIban = '';
		creditorBic = '';
		amount = '';
		reference = '';

		toast.success('Payment generated', `pain.001 file ready for ${newPaymentType === 'credit' ? 'Credit Transfer' : 'Direct Debit'}`);
	}

	async function downloadPain001(paymentId: string) {
		toast.success('Download started', 'pain.001.xml file is being downloaded');
	}
</script>

<svelte:head>
	<title>SEPA - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-7xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<h1 class="text-xl font-semibold text-[var(--color-ink)]">SEPA Payments</h1>
			<p class="text-sm text-[var(--color-ink-muted)]">
				Generate pain.001 credit transfers and pain.008 direct debits
			</p>
		</div>
		<Button onclick={() => { showNewPaymentModal = true; }}>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 5v14M5 12h14"/>
			</svg>
			New Payment
		</Button>
	</div>

	<!-- Filters -->
	<Card>
		<div class="flex flex-col sm:flex-row gap-4">
			<div class="flex-1">
				<Input
					type="search"
					bind:value={searchQuery}
					placeholder="Search payments..."
				/>
			</div>
			<div class="flex gap-3">
				<select bind:value={selectedType} class="input h-10 w-40">
					<option value="all">All types</option>
					<option value="credit">Credit Transfer</option>
					<option value="debit">Direct Debit</option>
				</select>
				<select bind:value={selectedStatus} class="input h-10 w-32">
					<option value="all">All status</option>
					<option value="draft">Draft</option>
					<option value="generated">Generated</option>
					<option value="sent">Sent</option>
					<option value="executed">Executed</option>
				</select>
			</div>
		</div>
	</Card>

	<!-- Payments list -->
	{#if filteredPayments.length === 0}
		<EmptyState
			title="No payments found"
			description="Create your first SEPA payment to get started."
			onAction={() => { showNewPaymentModal = true; }}
			actionLabel="Create Payment"
		>
			{#snippet icon()}
				<svg class="w-12 h-12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<rect width="20" height="14" x="2" y="5" rx="2"/>
					<line x1="2" x2="22" y1="10" y2="10"/>
				</svg>
			{/snippet}
		</EmptyState>
	{:else}
		<Card padding="none">
			<table class="table">
				<thead>
					<tr>
						<th>Reference</th>
						<th>Type</th>
						<th>Beneficiary</th>
						<th>IBAN</th>
						<th>Amount</th>
						<th>Execution</th>
						<th>Status</th>
						<th class="w-24"></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredPayments as payment}
						<tr>
							<td>
								<span class="font-mono text-sm">{payment.reference}</span>
							</td>
							<td>
								<span class="text-xs font-medium px-2 py-1 rounded bg-[var(--color-paper-inset)] text-[var(--color-ink-secondary)]">
									{payment.type === 'credit' ? 'CT' : 'DD'}
								</span>
							</td>
							<td>
								<span class="text-sm text-[var(--color-ink)]">{payment.creditor}</span>
							</td>
							<td>
								<span class="text-xs font-mono text-[var(--color-ink-muted)]">{formatIBAN(payment.iban)}</span>
							</td>
							<td>
								<span class="text-sm font-medium text-[var(--color-ink)]">
									{formatCurrency(payment.amount)}
								</span>
							</td>
							<td>
								<span class="text-sm text-[var(--color-ink-muted)]">
									{formatDate(payment.executionDate)}
								</span>
							</td>
							<td>
								<Badge variant={getStatusVariant(payment.status, 'payment')} size="sm">
									{getStatusLabel(payment.status, 'payment')}
								</Badge>
							</td>
							<td>
								<div class="flex items-center justify-end gap-1">
									{#if payment.status === 'generated' || payment.status === 'sent'}
										<Button variant="ghost" size="sm" onclick={() => downloadPain001(payment.id)}>
											<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
												<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
												<polyline points="7 10 12 15 17 10"/>
												<line x1="12" x2="12" y1="15" y2="3"/>
											</svg>
										</Button>
									{/if}
									<Button variant="ghost" size="sm">
										<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/>
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

<!-- New payment modal -->
{#if showNewPaymentModal}
	<div class="fixed inset-0 bg-black/50 backdrop-blur-sm z-[var(--z-modal)] flex items-center justify-center p-4">
		<div class="bg-[var(--color-paper-elevated)] rounded-xl shadow-2xl max-w-lg w-full p-6 animate-in max-h-[90vh] overflow-y-auto">
			<h3 class="text-lg font-semibold text-[var(--color-ink)]">New SEPA Payment</h3>
			<p class="text-sm text-[var(--color-ink-muted)] mt-1">
				Generate a pain.001 file for bank submission
			</p>

			<form onsubmit={(e) => { e.preventDefault(); handleGeneratePayment(); }} class="mt-6 space-y-4">
				<!-- Payment type -->
				<div role="group" aria-labelledby="payment-type-label">
					<span id="payment-type-label" class="label">Payment Type</span>
					<div class="grid grid-cols-2 gap-3">
						<button
							type="button"
							onclick={() => { newPaymentType = 'credit'; }}
							class={`p-3 rounded-lg border-2 text-left transition-all ${
								newPaymentType === 'credit'
									? 'border-[var(--color-accent)] bg-[var(--color-accent-muted)]'
									: 'border-black/10 hover:border-black/20'
							}`}
						>
							<p class={`font-medium ${newPaymentType === 'credit' ? 'text-[var(--color-accent)]' : 'text-[var(--color-ink)]'}`}>
								Credit Transfer
							</p>
							<p class="text-xs text-[var(--color-ink-muted)]">pain.001 - Send money</p>
						</button>
						<button
							type="button"
							onclick={() => { newPaymentType = 'debit'; }}
							class={`p-3 rounded-lg border-2 text-left transition-all ${
								newPaymentType === 'debit'
									? 'border-[var(--color-accent)] bg-[var(--color-accent-muted)]'
									: 'border-black/10 hover:border-black/20'
							}`}
						>
							<p class={`font-medium ${newPaymentType === 'debit' ? 'text-[var(--color-accent)]' : 'text-[var(--color-ink)]'}`}>
								Direct Debit
							</p>
							<p class="text-xs text-[var(--color-ink-muted)]">pain.008 - Collect money</p>
						</button>
					</div>
				</div>

				<div>
					<label for="creditor-name" class="label">{newPaymentType === 'credit' ? 'Beneficiary Name' : 'Debtor Name'}</label>
					<Input type="text" id="creditor-name" bind:value={creditorName} placeholder="Company or person name" />
				</div>

				<div>
					<label for="creditor-iban" class="label">IBAN</label>
					<Input type="text" id="creditor-iban" bind:value={creditorIban} placeholder="AT61 1904 3002 3457 3201" />
				</div>

				<div>
					<label for="creditor-bic" class="label">BIC (optional)</label>
					<Input type="text" id="creditor-bic" bind:value={creditorBic} placeholder="GIBAATWWXXX" />
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="amount" class="label">Amount (EUR)</label>
						<Input type="number" id="amount" bind:value={amount} placeholder="0.00" step="0.01" min="0.01" />
					</div>
					<div>
						<label for="execution-date" class="label">Execution Date</label>
						<Input type="date" id="execution-date" bind:value={executionDate} />
					</div>
				</div>

				<div>
					<label for="reference" class="label">Reference (optional)</label>
					<Input type="text" id="reference" bind:value={reference} placeholder="Auto-generated if empty" />
				</div>

				<div class="flex justify-end gap-2 pt-2">
					<Button variant="secondary" type="button" onclick={() => { showNewPaymentModal = false; }}>Cancel</Button>
					<Button type="submit" loading={isGenerating}>Generate pain.001</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
