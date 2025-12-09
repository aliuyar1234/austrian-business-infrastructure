<script lang="ts">
	import { formatDate } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	type InvoiceFormat = 'xrechnung' | 'zugferd';
	type InvoiceStatus = 'draft' | 'generated' | 'sent';

	interface Invoice {
		id: string;
		invoiceNumber: string;
		customer: string;
		total: number;
		format: InvoiceFormat;
		status: InvoiceStatus;
		issueDate: Date;
		createdAt: Date;
	}

	// Mock data
	let invoices = $state<Invoice[]>([
		{
			id: '1',
			invoiceNumber: 'RE-2024-0042',
			customer: 'Stadtverwaltung Wien',
			total: 2450.00,
			format: 'xrechnung',
			status: 'sent',
			issueDate: new Date(Date.now() - 5 * 86400000),
			createdAt: new Date(Date.now() - 5 * 86400000),
		},
		{
			id: '2',
			invoiceNumber: 'RE-2024-0043',
			customer: 'Bundesministerium für Finanzen',
			total: 12800.00,
			format: 'xrechnung',
			status: 'generated',
			issueDate: new Date(Date.now() - 2 * 86400000),
			createdAt: new Date(Date.now() - 2 * 86400000),
		},
		{
			id: '3',
			invoiceNumber: 'RE-2024-0044',
			customer: 'Muster GmbH',
			total: 850.00,
			format: 'zugferd',
			status: 'draft',
			issueDate: new Date(),
			createdAt: new Date(),
		},
	]);

	let searchQuery = $state('');
	let selectedFormat = $state<InvoiceFormat | 'all'>('all');
	let selectedStatus = $state<InvoiceStatus | 'all'>('all');

	let filteredInvoices = $derived(
		invoices.filter(inv => {
			const matchesSearch = !searchQuery ||
				inv.invoiceNumber.toLowerCase().includes(searchQuery.toLowerCase()) ||
				inv.customer.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesFormat = selectedFormat === 'all' || inv.format === selectedFormat;
			const matchesStatus = selectedStatus === 'all' || inv.status === selectedStatus;
			return matchesSearch && matchesFormat && matchesStatus;
		})
	);

	function getStatusLabel(status: InvoiceStatus): string {
		switch (status) {
			case 'draft': return 'Draft';
			case 'generated': return 'Generated';
			case 'sent': return 'Sent';
		}
	}

	function getStatusVariant(status: InvoiceStatus): 'default' | 'warning' | 'success' {
		switch (status) {
			case 'draft': return 'default';
			case 'generated': return 'warning';
			case 'sent': return 'success';
		}
	}

	function getFormatLabel(format: InvoiceFormat): string {
		switch (format) {
			case 'xrechnung': return 'XRechnung';
			case 'zugferd': return 'ZUGFeRD';
		}
	}

	function formatCurrency(amount: number): string {
		return new Intl.NumberFormat('de-AT', {
			style: 'currency',
			currency: 'EUR',
		}).format(amount);
	}
</script>

<svelte:head>
	<title>Invoices - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-7xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<h1 class="text-xl font-semibold text-[var(--color-ink)]">E-Rechnung</h1>
			<p class="text-sm text-[var(--color-ink-muted)]">
				Create electronic invoices in XRechnung and ZUGFeRD format
			</p>
		</div>
		<Button href="/invoices/new">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 5v14M5 12h14"/>
			</svg>
			New Invoice
		</Button>
	</div>

	<!-- Filters -->
	<Card>
		<div class="flex flex-col sm:flex-row gap-4">
			<div class="flex-1">
				<Input
					type="search"
					bind:value={searchQuery}
					placeholder="Search invoices..."
				/>
			</div>
			<div class="flex gap-3">
				<select bind:value={selectedFormat} class="input h-10 w-36">
					<option value="all">All formats</option>
					<option value="xrechnung">XRechnung</option>
					<option value="zugferd">ZUGFeRD</option>
				</select>
				<select bind:value={selectedStatus} class="input h-10 w-32">
					<option value="all">All status</option>
					<option value="draft">Draft</option>
					<option value="generated">Generated</option>
					<option value="sent">Sent</option>
				</select>
			</div>
		</div>
	</Card>

	<!-- Invoices list -->
	{#if filteredInvoices.length === 0}
		<Card class="text-center py-12">
			<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
				<polyline points="14 2 14 8 20 8"/>
				<line x1="12" x2="12" y1="18" y2="12"/>
				<line x1="9" x2="15" y1="15" y2="15"/>
			</svg>
			<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">No invoices found</h3>
			<p class="mt-2 text-[var(--color-ink-muted)]">
				Create your first electronic invoice to get started.
			</p>
			<Button href="/invoices/new" class="mt-6">Create Invoice</Button>
		</Card>
	{:else}
		<Card padding="none">
			<table class="table">
				<thead>
					<tr>
						<th>Invoice</th>
						<th>Customer</th>
						<th>Format</th>
						<th>Total</th>
						<th>Date</th>
						<th>Status</th>
						<th class="w-20"></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredInvoices as invoice}
						<tr>
							<td>
								<span class="font-mono text-sm">{invoice.invoiceNumber}</span>
							</td>
							<td>
								<span class="text-sm text-[var(--color-ink)]">{invoice.customer}</span>
							</td>
							<td>
								<span class="text-xs font-medium px-2 py-1 rounded bg-[var(--color-paper-inset)] text-[var(--color-ink-secondary)]">
									{getFormatLabel(invoice.format)}
								</span>
							</td>
							<td>
								<span class="text-sm font-medium text-[var(--color-ink)]">
									{formatCurrency(invoice.total)}
								</span>
							</td>
							<td>
								<span class="text-sm text-[var(--color-ink-muted)]">
									{formatDate(invoice.issueDate)}
								</span>
							</td>
							<td>
								<Badge variant={getStatusVariant(invoice.status)} size="sm">
									{getStatusLabel(invoice.status)}
								</Badge>
							</td>
							<td>
								<div class="flex items-center justify-end gap-1">
									<Button variant="ghost" size="sm">
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
