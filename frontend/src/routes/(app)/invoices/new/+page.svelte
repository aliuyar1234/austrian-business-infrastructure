<script lang="ts">
	import { goto } from '$app/navigation';
	import { toast } from '$lib/stores/toast';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	type InvoiceFormat = 'xrechnung' | 'zugferd';

	interface LineItem {
		id: string;
		description: string;
		quantity: number;
		unit: string;
		unitPrice: number;
		vatRate: number;
	}

	// Form state
	let format = $state<InvoiceFormat>('xrechnung');
	let invoiceNumber = $state(`RE-${new Date().getFullYear()}-${String(Math.floor(Math.random() * 10000)).padStart(4, '0')}`);
	let issueDate = $state(new Date().toISOString().split('T')[0]);
	let dueDate = $state(new Date(Date.now() + 30 * 86400000).toISOString().split('T')[0]);

	// Supplier (pre-filled)
	let supplierName = $state('Meine Firma GmbH');
	let supplierStreet = $state('Musterstraße 1');
	let supplierPostcode = $state('1010');
	let supplierCity = $state('Wien');
	let supplierCountry = $state('AT');
	let supplierVatId = $state('ATU12345678');

	// Customer
	let customerName = $state('');
	let customerStreet = $state('');
	let customerPostcode = $state('');
	let customerCity = $state('');
	let customerCountry = $state('AT');
	let customerVatId = $state('');
	let leitwegId = $state(''); // For XRechnung

	// Line items
	let lineItems = $state<LineItem[]>([
		{ id: '1', description: '', quantity: 1, unit: 'Stück', unitPrice: 0, vatRate: 20 }
	]);

	let isGenerating = $state(false);

	// Calculations
	let subtotal = $derived(
		lineItems.reduce((sum, item) => sum + (item.quantity * item.unitPrice), 0)
	);

	let vatAmount = $derived(
		lineItems.reduce((sum, item) => sum + (item.quantity * item.unitPrice * item.vatRate / 100), 0)
	);

	let total = $derived(subtotal + vatAmount);

	function addLineItem() {
		lineItems = [
			...lineItems,
			{ id: String(Date.now()), description: '', quantity: 1, unit: 'Stück', unitPrice: 0, vatRate: 20 }
		];
	}

	function removeLineItem(id: string) {
		if (lineItems.length > 1) {
			lineItems = lineItems.filter(item => item.id !== id);
		}
	}

	function formatCurrency(amount: number): string {
		return new Intl.NumberFormat('de-AT', {
			style: 'currency',
			currency: 'EUR',
		}).format(amount);
	}

	async function handleGenerate() {
		// Basic validation
		if (!customerName.trim()) {
			toast.error('Validation error', 'Customer name is required');
			return;
		}
		if (!lineItems.some(item => item.description.trim())) {
			toast.error('Validation error', 'At least one line item is required');
			return;
		}
		if (format === 'xrechnung' && !leitwegId.trim()) {
			toast.error('Validation error', 'Leitweg-ID is required for XRechnung');
			return;
		}

		isGenerating = true;

		// Simulate API call
		await new Promise(r => setTimeout(r, 2000));

		isGenerating = false;
		toast.success('Invoice generated', `${invoiceNumber} has been created in ${format === 'xrechnung' ? 'XRechnung' : 'ZUGFeRD'} format`);
		goto('/invoices');
	}
</script>

<svelte:head>
	<title>New Invoice - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<a href="/invoices" aria-label="Back to invoices" class="w-10 h-10 flex items-center justify-center rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors">
			<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m12 19-7-7 7-7M19 12H5"/>
			</svg>
		</a>
		<div>
			<h1 class="text-xl font-semibold text-[var(--color-ink)]">Create Invoice</h1>
			<p class="text-sm text-[var(--color-ink-muted)]">
				Generate a new electronic invoice
			</p>
		</div>
	</div>

	<!-- Format selection -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Invoice Format</h2>
		<div class="grid sm:grid-cols-2 gap-4">
			<button
				onclick={() => { format = 'xrechnung'; }}
				class={`p-4 rounded-lg border-2 text-left transition-all ${
					format === 'xrechnung'
						? 'border-[var(--color-accent)] bg-[var(--color-accent-muted)]'
						: 'border-black/10 hover:border-black/20'
				}`}
			>
				<div class="flex items-center gap-3">
					<div class={`w-10 h-10 rounded-lg flex items-center justify-center ${
						format === 'xrechnung' ? 'bg-[var(--color-accent)]/10' : 'bg-[var(--color-paper-inset)]'
					}`}>
						<span class={`text-sm font-bold ${format === 'xrechnung' ? 'text-[var(--color-accent)]' : 'text-[var(--color-ink-muted)]'}`}>XR</span>
					</div>
					<div>
						<p class={`font-medium ${format === 'xrechnung' ? 'text-[var(--color-accent)]' : 'text-[var(--color-ink)]'}`}>XRechnung</p>
						<p class="text-xs text-[var(--color-ink-muted)]">For public sector (B2G)</p>
					</div>
				</div>
			</button>

			<button
				onclick={() => { format = 'zugferd'; }}
				class={`p-4 rounded-lg border-2 text-left transition-all ${
					format === 'zugferd'
						? 'border-[var(--color-accent)] bg-[var(--color-accent-muted)]'
						: 'border-black/10 hover:border-black/20'
				}`}
			>
				<div class="flex items-center gap-3">
					<div class={`w-10 h-10 rounded-lg flex items-center justify-center ${
						format === 'zugferd' ? 'bg-[var(--color-accent)]/10' : 'bg-[var(--color-paper-inset)]'
					}`}>
						<span class={`text-sm font-bold ${format === 'zugferd' ? 'text-[var(--color-accent)]' : 'text-[var(--color-ink-muted)]'}`}>ZF</span>
					</div>
					<div>
						<p class={`font-medium ${format === 'zugferd' ? 'text-[var(--color-accent)]' : 'text-[var(--color-ink)]'}`}>ZUGFeRD</p>
						<p class="text-xs text-[var(--color-ink-muted)]">PDF/A with XML (B2B)</p>
					</div>
				</div>
			</button>
		</div>
	</Card>

	<!-- Invoice details -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Invoice Details</h2>
		<div class="grid sm:grid-cols-3 gap-4">
			<div>
				<label for="invoiceNumber" class="label">Invoice Number</label>
				<Input type="text" id="invoiceNumber" bind:value={invoiceNumber} />
			</div>
			<div>
				<label for="issueDate" class="label">Issue Date</label>
				<Input type="date" id="issueDate" bind:value={issueDate} />
			</div>
			<div>
				<label for="dueDate" class="label">Due Date</label>
				<Input type="date" id="dueDate" bind:value={dueDate} />
			</div>
		</div>
	</Card>

	<!-- Customer -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Customer</h2>
		<div class="space-y-4">
			<div class="grid sm:grid-cols-2 gap-4">
				<div>
					<label for="customerName" class="label">Name / Company</label>
					<Input type="text" id="customerName" bind:value={customerName} placeholder="Customer name" />
				</div>
				<div>
					<label for="customerVatId" class="label">VAT ID (optional)</label>
					<Input type="text" id="customerVatId" bind:value={customerVatId} placeholder="ATU..." />
				</div>
			</div>
			<div>
				<label for="customerStreet" class="label">Street</label>
				<Input type="text" id="customerStreet" bind:value={customerStreet} placeholder="Street address" />
			</div>
			<div class="grid sm:grid-cols-3 gap-4">
				<div>
					<label for="customerPostcode" class="label">Postcode</label>
					<Input type="text" id="customerPostcode" bind:value={customerPostcode} placeholder="1010" />
				</div>
				<div>
					<label for="customerCity" class="label">City</label>
					<Input type="text" id="customerCity" bind:value={customerCity} placeholder="Vienna" />
				</div>
				<div>
					<label for="customerCountry" class="label">Country</label>
					<select id="customerCountry" bind:value={customerCountry} class="input h-10">
						<option value="AT">Austria</option>
						<option value="DE">Germany</option>
						<option value="CH">Switzerland</option>
					</select>
				</div>
			</div>
			{#if format === 'xrechnung'}
				<div>
					<label for="leitwegId" class="label">Leitweg-ID (required for XRechnung)</label>
					<Input type="text" id="leitwegId" bind:value={leitwegId} placeholder="e.g., 991-12345-67" />
					<p class="mt-1 text-xs text-[var(--color-ink-muted)]">The routing ID of the public sector recipient</p>
				</div>
			{/if}
		</div>
	</Card>

	<!-- Line items -->
	<Card>
		<div class="flex items-center justify-between mb-4">
			<h2 class="font-semibold text-[var(--color-ink)]">Line Items</h2>
			<Button variant="secondary" size="sm" onclick={addLineItem}>
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 5v14M5 12h14"/>
				</svg>
				Add item
			</Button>
		</div>

		<div class="space-y-4">
			{#each lineItems as item, i}
				<div class="p-4 rounded-lg bg-[var(--color-paper-inset)] space-y-3">
					<div class="flex items-center justify-between">
						<span class="text-sm font-medium text-[var(--color-ink-muted)]">Item {i + 1}</span>
						{#if lineItems.length > 1}
							<button
								onclick={() => removeLineItem(item.id)}
								class="text-sm text-[var(--color-error)] hover:underline"
							>
								Remove
							</button>
						{/if}
					</div>

					<div>
						<label for="desc-{item.id}" class="label">Description</label>
						<Input type="text" id="desc-{item.id}" bind:value={item.description} placeholder="Item description" />
					</div>

					<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
						<div>
							<label for="qty-{item.id}" class="label">Quantity</label>
							<Input type="number" id="qty-{item.id}" bind:value={item.quantity} min="1" step="1" />
						</div>
						<div>
							<label for="unit-{item.id}" class="label">Unit</label>
							<select id="unit-{item.id}" bind:value={item.unit} class="input h-10">
								<option value="Stück">Stück</option>
								<option value="Stunde">Stunde</option>
								<option value="Tag">Tag</option>
								<option value="Pauschal">Pauschal</option>
							</select>
						</div>
						<div>
							<label for="price-{item.id}" class="label">Unit Price (€)</label>
							<Input type="number" id="price-{item.id}" bind:value={item.unitPrice} min="0" step="0.01" />
						</div>
						<div>
							<label for="vat-{item.id}" class="label">VAT Rate</label>
							<select id="vat-{item.id}" bind:value={item.vatRate} class="input h-10">
								<option value={20}>20%</option>
								<option value={10}>10%</option>
								<option value={13}>13%</option>
								<option value={0}>0%</option>
							</select>
						</div>
					</div>

					<div class="text-right text-sm">
						<span class="text-[var(--color-ink-muted)]">Line total: </span>
						<span class="font-medium text-[var(--color-ink)]">
							{formatCurrency(item.quantity * item.unitPrice)}
						</span>
					</div>
				</div>
			{/each}
		</div>
	</Card>

	<!-- Summary -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Summary</h2>
		<div class="space-y-3">
			<div class="flex items-center justify-between py-2">
				<span class="text-sm text-[var(--color-ink-secondary)]">Subtotal</span>
				<span class="text-sm font-medium text-[var(--color-ink)]">{formatCurrency(subtotal)}</span>
			</div>
			<div class="flex items-center justify-between py-2">
				<span class="text-sm text-[var(--color-ink-secondary)]">VAT</span>
				<span class="text-sm font-medium text-[var(--color-ink)]">{formatCurrency(vatAmount)}</span>
			</div>
			<div class="flex items-center justify-between py-3 border-t border-black/10">
				<span class="font-medium text-[var(--color-ink)]">Total</span>
				<span class="text-lg font-bold text-[var(--color-ink)]">{formatCurrency(total)}</span>
			</div>
		</div>
	</Card>

	<!-- Actions -->
	<div class="flex items-center justify-end gap-3">
		<Button variant="secondary" href="/invoices">Cancel</Button>
		<Button onclick={handleGenerate} loading={isGenerating}>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
				<polyline points="14 2 14 8 20 8"/>
			</svg>
			Generate Invoice
		</Button>
	</div>
</div>
