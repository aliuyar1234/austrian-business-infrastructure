<script lang="ts">
	import { goto } from '$app/navigation';
	import { formatCurrency } from '$lib/utils';
	import { toast } from '$lib/stores/toast';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { z } from 'zod';

	// Mock accounts
	const accounts = [
		{ id: '1', name: 'Muster GmbH', teilnehmerId: '123456789' },
		{ id: '2', name: 'Test AG', teilnehmerId: '987654321' },
	];

	// Current month/year for defaults
	const now = new Date();
	const currentMonth = now.getMonth(); // 0-11
	const currentYear = now.getFullYear();

	// Form state
	let selectedAccountId = $state('');
	let year = $state(currentYear);
	let month = $state(currentMonth === 0 ? 12 : currentMonth); // Previous month default

	// UVA Kennzahlen (simplified)
	let kz000 = $state(''); // Steuerpflichtige Umsätze 20%
	let kz001 = $state(''); // Steuerpflichtige Umsätze 10%
	let kz022 = $state(''); // Erwerbsteuer
	let kz060 = $state(''); // Abziehbare Vorsteuer
	let kz065 = $state(''); // Vorsteuer aus EUSt
	let kz070 = $state(''); // Einfuhrumsatzsteuer

	let isValidating = $state(false);
	let isSubmitting = $state(false);
	let validationResult = $state<{ valid: boolean; errors: string[] } | null>(null);
	let errors = $state<Record<string, string>>({});

	// Calculate derived values
	let ust20 = $derived(parseFloat(kz000 || '0') * 0.20);
	let ust10 = $derived(parseFloat(kz001 || '0') * 0.10);
	let totalUst = $derived(ust20 + ust10 + parseFloat(kz022 || '0'));
	let totalVst = $derived(parseFloat(kz060 || '0') + parseFloat(kz065 || '0') + parseFloat(kz070 || '0'));
	let zahllast = $derived(totalUst - totalVst);

	const months = [
		'Jänner', 'Februar', 'März', 'April', 'Mai', 'Juni',
		'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'
	];

	async function handleValidate() {
		if (!selectedAccountId) {
			errors = { account: 'Please select an account' };
			return;
		}

		isValidating = true;
		validationResult = null;

		// Simulate API validation
		await new Promise(r => setTimeout(r, 1000));

		const validationErrors: string[] = [];
		if (parseFloat(kz000 || '0') < 0) validationErrors.push('KZ000: Umsätze cannot be negative');
		if (parseFloat(kz060 || '0') < 0) validationErrors.push('KZ060: Vorsteuer cannot be negative');

		validationResult = {
			valid: validationErrors.length === 0,
			errors: validationErrors,
		};

		isValidating = false;

		if (validationResult.valid) {
			toast.success('Validation passed', 'UVA is ready for submission');
		} else {
			toast.error('Validation failed', `${validationErrors.length} error(s) found`);
		}
	}

	async function handleSubmit() {
		if (!validationResult?.valid) {
			toast.warning('Please validate first', 'Run validation before submitting');
			return;
		}

		isSubmitting = true;

		// Simulate API submission
		await new Promise(r => setTimeout(r, 2000));

		isSubmitting = false;
		toast.success('UVA submitted successfully', `Reference: FO-${year}-${Math.random().toString().slice(2, 9)}`);
		goto('/uva');
	}
</script>

<svelte:head>
	<title>New UVA - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<a href="/uva" aria-label="Back to UVA" class="w-10 h-10 flex items-center justify-center rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors">
			<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m12 19-7-7 7-7M19 12H5"/>
			</svg>
		</a>
		<div>
			<h1 class="text-xl font-semibold text-[var(--color-ink)]">Create UVA</h1>
			<p class="text-sm text-[var(--color-ink-muted)]">
				Fill in the VAT return details
			</p>
		</div>
	</div>

	<!-- Account & Period selection -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Period Selection</h2>
		<div class="grid sm:grid-cols-3 gap-4">
			<div>
				<label for="account" class="label">Account</label>
				<select id="account" bind:value={selectedAccountId} class="input h-10">
					<option value="">Select account...</option>
					{#each accounts as acc}
						<option value={acc.id}>{acc.name}</option>
					{/each}
				</select>
				{#if errors.account}
					<p class="mt-1 text-xs text-[var(--color-error)]">{errors.account}</p>
				{/if}
			</div>
			<div>
				<label for="month" class="label">Month</label>
				<select id="month" bind:value={month} class="input h-10">
					{#each months as m, i}
						<option value={i + 1}>{m}</option>
					{/each}
				</select>
			</div>
			<div>
				<label for="year" class="label">Year</label>
				<select id="year" bind:value={year} class="input h-10">
					{#each [currentYear, currentYear - 1, currentYear - 2] as y}
						<option value={y}>{y}</option>
					{/each}
				</select>
			</div>
		</div>
	</Card>

	<!-- Kennzahlen form -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Kennzahlen</h2>

		<!-- Steuerpflichtige Umsätze -->
		<div class="space-y-4">
			<div class="pb-4 border-b border-black/6">
				<h3 class="text-sm font-medium text-[var(--color-ink-secondary)] mb-3">Steuerpflichtige Umsätze</h3>
				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="kz000" class="label">KZ 000 - Umsätze 20%</label>
						<Input type="number" id="kz000" bind:value={kz000} placeholder="0.00" step="0.01" />
						{#if kz000}
							<p class="mt-1 text-xs text-[var(--color-ink-muted)]">USt: {formatCurrency(ust20)}</p>
						{/if}
					</div>
					<div>
						<label for="kz001" class="label">KZ 001 - Umsätze 10%</label>
						<Input type="number" id="kz001" bind:value={kz001} placeholder="0.00" step="0.01" />
						{#if kz001}
							<p class="mt-1 text-xs text-[var(--color-ink-muted)]">USt: {formatCurrency(ust10)}</p>
						{/if}
					</div>
				</div>
			</div>

			<div class="pb-4 border-b border-black/6">
				<h3 class="text-sm font-medium text-[var(--color-ink-secondary)] mb-3">Erwerbsteuer</h3>
				<div class="grid sm:grid-cols-2 gap-4">
					<div>
						<label for="kz022" class="label">KZ 022 - Erwerbsteuer</label>
						<Input type="number" id="kz022" bind:value={kz022} placeholder="0.00" step="0.01" />
					</div>
				</div>
			</div>

			<div class="pb-4 border-b border-black/6">
				<h3 class="text-sm font-medium text-[var(--color-ink-secondary)] mb-3">Vorsteuer</h3>
				<div class="grid sm:grid-cols-3 gap-4">
					<div>
						<label for="kz060" class="label">KZ 060 - Vorsteuer</label>
						<Input type="number" id="kz060" bind:value={kz060} placeholder="0.00" step="0.01" />
					</div>
					<div>
						<label for="kz065" class="label">KZ 065 - Vorsteuer EUSt</label>
						<Input type="number" id="kz065" bind:value={kz065} placeholder="0.00" step="0.01" />
					</div>
					<div>
						<label for="kz070" class="label">KZ 070 - EUSt</label>
						<Input type="number" id="kz070" bind:value={kz070} placeholder="0.00" step="0.01" />
					</div>
				</div>
			</div>
		</div>
	</Card>

	<!-- Summary -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Summary</h2>
		<div class="space-y-3">
			<div class="flex items-center justify-between py-2">
				<span class="text-sm text-[var(--color-ink-secondary)]">Total USt</span>
				<span class="text-sm font-medium text-[var(--color-ink)]">{formatCurrency(totalUst)}</span>
			</div>
			<div class="flex items-center justify-between py-2">
				<span class="text-sm text-[var(--color-ink-secondary)]">Total Vorsteuer</span>
				<span class="text-sm font-medium text-[var(--color-ink)]">- {formatCurrency(totalVst)}</span>
			</div>
			<div class="flex items-center justify-between py-3 border-t border-black/10">
				<span class="font-medium text-[var(--color-ink)]">Zahllast / Gutschrift</span>
				<span class={`text-lg font-bold ${zahllast >= 0 ? 'text-[var(--color-error)]' : 'text-[var(--color-success)]'}`}>
					{formatCurrency(zahllast)}
				</span>
			</div>
		</div>
	</Card>

	<!-- Validation result -->
	{#if validationResult}
		<div class={`flex gap-3 p-4 rounded-lg ${validationResult.valid ? 'bg-[var(--color-success-muted)]' : 'bg-[var(--color-error-muted)]'}`}>
			<svg class={`w-5 h-5 flex-shrink-0 mt-0.5 ${validationResult.valid ? 'text-[var(--color-success)]' : 'text-[var(--color-error)]'}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				{#if validationResult.valid}
					<polyline points="20 6 9 17 4 12"/>
				{:else}
					<circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/>
				{/if}
			</svg>
			<div class={`text-sm ${validationResult.valid ? 'text-[var(--color-success)]' : 'text-[var(--color-error)]'}`}>
				{#if validationResult.valid}
					<p class="font-medium">Validation passed</p>
					<p class="mt-0.5 opacity-80">UVA is ready for submission to FinanzOnline</p>
				{:else}
					<p class="font-medium">Validation failed</p>
					<ul class="mt-1 space-y-1 opacity-80">
						{#each validationResult.errors as error}
							<li>• {error}</li>
						{/each}
					</ul>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Actions -->
	<div class="flex items-center justify-end gap-3">
		<Button variant="secondary" href="/uva">Cancel</Button>
		<Button variant="secondary" onclick={handleValidate} loading={isValidating}>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<polyline points="20 6 9 17 4 12"/>
			</svg>
			Validate
		</Button>
		<Button onclick={handleSubmit} loading={isSubmitting} disabled={!validationResult?.valid}>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M22 2 11 13M22 2l-7 20-4-9-9-4 20-7z"/>
			</svg>
			Submit to FinanzOnline
		</Button>
	</div>
</div>
