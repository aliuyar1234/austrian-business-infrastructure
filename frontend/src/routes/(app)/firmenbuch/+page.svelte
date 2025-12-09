<script lang="ts">
	import { formatDate } from '$lib/utils';
	import { toast } from '$lib/stores/toast';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	type CompanyStatus = 'active' | 'dissolved' | 'liquidating' | 'unknown';
	type LegalForm = 'GmbH' | 'AG' | 'OG' | 'KG' | 'e.U.' | 'GesbR' | 'Verein' | 'Other';

	interface Company {
		fn: string; // Firmenbuchnummer
		name: string;
		legalForm: LegalForm;
		address: string;
		status: CompanyStatus;
		registrationDate: Date;
		capital?: number;
		directors?: string[];
	}

	// Search state
	let searchQuery = $state('');
	let searchType = $state<'name' | 'fn'>('name');
	let isSearching = $state(false);
	let hasSearched = $state(false);

	// Results
	let results = $state<Company[]>([]);

	// Selected company for detail view
	let selectedCompany = $state<Company | null>(null);
	let isLoadingExtract = $state(false);

	// Watchlist
	let watchlist = $state<string[]>(['FN 123456 a', 'FN 789012 b']);

	// Mock search results
	const mockResults: Company[] = [
		{
			fn: 'FN 123456 a',
			name: 'Muster GmbH',
			legalForm: 'GmbH',
			address: 'Musterstraße 1, 1010 Wien',
			status: 'active',
			registrationDate: new Date(2015, 3, 15),
			capital: 35000,
			directors: ['Johann Mustermann', 'Maria Musterfrau'],
		},
		{
			fn: 'FN 234567 b',
			name: 'Test Handels AG',
			legalForm: 'AG',
			address: 'Testgasse 42, 8010 Graz',
			status: 'active',
			registrationDate: new Date(2010, 8, 1),
			capital: 70000,
			directors: ['Thomas Tester'],
		},
		{
			fn: 'FN 345678 c',
			name: 'Demo & Partner OG',
			legalForm: 'OG',
			address: 'Demoplatz 7, 5020 Salzburg',
			status: 'active',
			registrationDate: new Date(2020, 0, 10),
			directors: ['Anna Demo', 'Peter Partner'],
		},
		{
			fn: 'FN 456789 d',
			name: 'Alt GmbH in Liquidation',
			legalForm: 'GmbH',
			address: 'Altweg 3, 4020 Linz',
			status: 'liquidating',
			registrationDate: new Date(2005, 5, 20),
			capital: 35000,
			directors: ['Liquidator Müller'],
		},
	];

	async function handleSearch() {
		if (!searchQuery.trim()) {
			toast.error('Please enter a search term');
			return;
		}

		isSearching = true;
		hasSearched = true;
		await new Promise(r => setTimeout(r, 1000));

		// Filter mock results based on search
		const q = searchQuery.toLowerCase();
		if (searchType === 'fn') {
			results = mockResults.filter(c => c.fn.toLowerCase().includes(q));
		} else {
			results = mockResults.filter(c => c.name.toLowerCase().includes(q));
		}

		isSearching = false;

		if (results.length === 0) {
			toast.info('No results', 'No companies found matching your search');
		}
	}

	async function viewExtract(company: Company) {
		selectedCompany = company;
		isLoadingExtract = true;
		await new Promise(r => setTimeout(r, 800));
		isLoadingExtract = false;
	}

	function closeExtract() {
		selectedCompany = null;
	}

	function toggleWatchlist(fn: string) {
		if (watchlist.includes(fn)) {
			watchlist = watchlist.filter(f => f !== fn);
			toast.info('Removed from watchlist');
		} else {
			watchlist = [...watchlist, fn];
			toast.success('Added to watchlist', 'You will be notified of changes');
		}
	}

	function isInWatchlist(fn: string): boolean {
		return watchlist.includes(fn);
	}

	function getStatusLabel(status: CompanyStatus): string {
		switch (status) {
			case 'active': return 'Active';
			case 'dissolved': return 'Dissolved';
			case 'liquidating': return 'In Liquidation';
			case 'unknown': return 'Unknown';
		}
	}

	function getStatusVariant(status: CompanyStatus): 'success' | 'error' | 'warning' | 'default' {
		switch (status) {
			case 'active': return 'success';
			case 'dissolved': return 'error';
			case 'liquidating': return 'warning';
			case 'unknown': return 'default';
		}
	}

	function formatCurrency(amount: number): string {
		return new Intl.NumberFormat('de-AT', {
			style: 'currency',
			currency: 'EUR',
			minimumFractionDigits: 0,
		}).format(amount);
	}

	async function downloadExtract() {
		toast.success('Download started', 'Firmenbuchauszug PDF is being downloaded');
	}

	async function orderCertifiedExtract() {
		toast.info('Order placed', 'Certified extract will be delivered within 2 business days');
	}
</script>

<svelte:head>
	<title>Firmenbuch - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-6xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div>
		<h1 class="text-xl font-semibold text-[var(--color-ink)]">Firmenbuch</h1>
		<p class="text-sm text-[var(--color-ink-muted)]">
			Search the Austrian Commercial Register
		</p>
	</div>

	<!-- Search -->
	<Card>
		<form onsubmit={(e) => { e.preventDefault(); handleSearch(); }} class="space-y-4">
			<div class="flex flex-col sm:flex-row gap-4">
				<div class="flex-1">
					<label for="search-query" class="label">Search</label>
					<Input
						type="text"
						id="search-query"
						bind:value={searchQuery}
						placeholder={searchType === 'fn' ? 'FN 123456 a' : 'Company name...'}
					/>
				</div>
				<div class="w-full sm:w-48">
					<label for="search-type" class="label">Search by</label>
					<select id="search-type" bind:value={searchType} class="input h-10">
						<option value="name">Company Name</option>
						<option value="fn">Firmenbuchnummer</option>
					</select>
				</div>
				<div class="flex items-end">
					<Button type="submit" loading={isSearching} class="w-full sm:w-auto">
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="11" cy="11" r="8"/>
							<path d="m21 21-4.35-4.35"/>
						</svg>
						Search
					</Button>
				</div>
			</div>
		</form>
	</Card>

	<div class="grid lg:grid-cols-3 gap-6">
		<!-- Results -->
		<div class="lg:col-span-2 space-y-4">
			{#if !hasSearched}
				<Card class="text-center py-12">
					<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
						<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/>
						<circle cx="12" cy="7" r="4"/>
						<path d="M22 21v-2a4 4 0 0 0-3-3.87"/>
						<path d="M16 3.13a4 4 0 0 1 0 7.75"/>
					</svg>
					<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">Search the Firmenbuch</h3>
					<p class="mt-2 text-[var(--color-ink-muted)]">
						Enter a company name or Firmenbuchnummer to search.
					</p>
				</Card>
			{:else if isSearching}
				<Card class="text-center py-12">
					<div class="w-8 h-8 mx-auto border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
					<p class="mt-4 text-[var(--color-ink-muted)]">Searching...</p>
				</Card>
			{:else if results.length === 0}
				<Card class="text-center py-12">
					<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
						<circle cx="11" cy="11" r="8"/>
						<path d="m21 21-4.35-4.35"/>
						<path d="M8 8l6 6M14 8l-6 6"/>
					</svg>
					<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">No results found</h3>
					<p class="mt-2 text-[var(--color-ink-muted)]">
						Try a different search term or search type.
					</p>
				</Card>
			{:else}
				<div class="space-y-3">
					{#each results as company}
						<Card class="hover:shadow-md transition-shadow">
							<div class="flex items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-2 flex-wrap">
										<h3 class="font-semibold text-[var(--color-ink)]">{company.name}</h3>
										<Badge variant={getStatusVariant(company.status)} size="sm">
											{getStatusLabel(company.status)}
										</Badge>
									</div>
									<p class="text-sm font-mono text-[var(--color-ink-secondary)] mt-1">{company.fn}</p>
									<p class="text-sm text-[var(--color-ink-muted)] mt-1">{company.address}</p>
									<div class="flex items-center gap-4 mt-2 text-xs text-[var(--color-ink-muted)]">
										<span>{company.legalForm}</span>
										{#if company.capital}
											<span>Capital: {formatCurrency(company.capital)}</span>
										{/if}
										<span>Reg: {formatDate(company.registrationDate)}</span>
									</div>
								</div>
								<div class="flex items-center gap-2">
									<Button
										variant="ghost"
										size="sm"
										onclick={() => toggleWatchlist(company.fn)}
										aria-label={isInWatchlist(company.fn) ? 'Remove from watchlist' : 'Add to watchlist'}
									>
										<svg
											class="w-4 h-4"
											viewBox="0 0 24 24"
											fill={isInWatchlist(company.fn) ? 'currentColor' : 'none'}
											stroke="currentColor"
											stroke-width="2"
										>
											<path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/>
										</svg>
									</Button>
									<Button size="sm" onclick={() => viewExtract(company)}>
										View Extract
									</Button>
								</div>
							</div>
						</Card>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Sidebar: Watchlist -->
		<div class="space-y-4">
			<Card>
				<h3 class="font-semibold text-[var(--color-ink)] mb-3">
					<div class="flex items-center gap-2">
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/>
						</svg>
						Watchlist
					</div>
				</h3>
				{#if watchlist.length === 0}
					<p class="text-sm text-[var(--color-ink-muted)]">
						No companies in your watchlist. Add companies to monitor for changes.
					</p>
				{:else}
					<div class="space-y-2">
						{#each watchlist as fn}
							<div class="flex items-center justify-between p-2 rounded-lg bg-[var(--color-paper-inset)]">
								<span class="text-sm font-mono text-[var(--color-ink-secondary)]">{fn}</span>
								<button
									onclick={() => toggleWatchlist(fn)}
									class="text-[var(--color-ink-muted)] hover:text-[var(--color-error)] transition-colors"
									aria-label="Remove from watchlist"
								>
									<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<path d="M18 6 6 18M6 6l12 12"/>
									</svg>
								</button>
							</div>
						{/each}
					</div>
					<p class="mt-3 text-xs text-[var(--color-ink-muted)]">
						You'll be notified when changes are registered for watched companies.
					</p>
				{/if}
			</Card>

			<Card>
				<h3 class="font-semibold text-[var(--color-ink)] mb-3">About Firmenbuch</h3>
				<div class="text-sm text-[var(--color-ink-muted)] space-y-2">
					<p>
						The Firmenbuch (Commercial Register) contains information about all registered companies in Austria.
					</p>
					<p>
						Data includes company name, legal form, registered office, capital, directors, and shareholders.
					</p>
				</div>
			</Card>
		</div>
	</div>
</div>

<!-- Company Extract Modal -->
{#if selectedCompany}
	<div class="fixed inset-0 bg-black/50 backdrop-blur-sm z-[var(--z-modal)] flex items-center justify-center p-4">
		<div class="bg-[var(--color-paper-elevated)] rounded-xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden animate-in">
			<!-- Header -->
			<div class="p-4 border-b border-black/6 flex items-center justify-between">
				<div>
					<h3 class="text-lg font-semibold text-[var(--color-ink)]">{selectedCompany.name}</h3>
					<p class="text-sm font-mono text-[var(--color-ink-muted)]">{selectedCompany.fn}</p>
				</div>
				<button onclick={closeExtract} class="p-2 rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors" aria-label="Close">
					<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M18 6 6 18M6 6l12 12"/>
					</svg>
				</button>
			</div>

			<!-- Content -->
			<div class="p-6 overflow-y-auto max-h-[60vh]">
				{#if isLoadingExtract}
					<div class="text-center py-8">
						<div class="w-8 h-8 mx-auto border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin"></div>
						<p class="mt-4 text-[var(--color-ink-muted)]">Loading extract...</p>
					</div>
				{:else}
					<div class="space-y-6">
						<!-- Status -->
						<div class="flex items-center gap-2">
							<Badge variant={getStatusVariant(selectedCompany.status)} size="sm">
								{getStatusLabel(selectedCompany.status)}
							</Badge>
							<span class="text-sm text-[var(--color-ink-muted)]">
								Registered {formatDate(selectedCompany.registrationDate)}
							</span>
						</div>

						<!-- Basic Info -->
						<div class="grid sm:grid-cols-2 gap-4">
							<div>
								<p class="text-xs text-[var(--color-ink-muted)] uppercase tracking-wide">Legal Form</p>
								<p class="text-sm font-medium text-[var(--color-ink)] mt-1">{selectedCompany.legalForm}</p>
							</div>
							{#if selectedCompany.capital}
								<div>
									<p class="text-xs text-[var(--color-ink-muted)] uppercase tracking-wide">Share Capital</p>
									<p class="text-sm font-medium text-[var(--color-ink)] mt-1">{formatCurrency(selectedCompany.capital)}</p>
								</div>
							{/if}
						</div>

						<!-- Address -->
						<div>
							<p class="text-xs text-[var(--color-ink-muted)] uppercase tracking-wide">Registered Office</p>
							<p class="text-sm font-medium text-[var(--color-ink)] mt-1">{selectedCompany.address}</p>
						</div>

						<!-- Directors -->
						{#if selectedCompany.directors && selectedCompany.directors.length > 0}
							<div>
								<p class="text-xs text-[var(--color-ink-muted)] uppercase tracking-wide">Directors / Managing Partners</p>
								<ul class="mt-2 space-y-1">
									{#each selectedCompany.directors as director}
										<li class="flex items-center gap-2">
											<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
												<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
												<circle cx="12" cy="7" r="4"/>
											</svg>
											<span class="text-sm text-[var(--color-ink)]">{director}</span>
										</li>
									{/each}
								</ul>
							</div>
						{/if}

						<!-- Note -->
						<div class="p-3 rounded-lg bg-[var(--color-paper-inset)] text-sm text-[var(--color-ink-muted)]">
							<p>This is a summary view. For complete legal documentation, order a certified extract.</p>
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="p-4 border-t border-black/6 flex items-center justify-between">
				<Button
					variant="ghost"
					size="sm"
					onclick={() => toggleWatchlist(selectedCompany!.fn)}
				>
					<svg
						class="w-4 h-4"
						viewBox="0 0 24 24"
						fill={isInWatchlist(selectedCompany.fn) ? 'currentColor' : 'none'}
						stroke="currentColor"
						stroke-width="2"
					>
						<path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/>
					</svg>
					{isInWatchlist(selectedCompany.fn) ? 'In Watchlist' : 'Add to Watchlist'}
				</Button>
				<div class="flex items-center gap-2">
					<Button variant="secondary" onclick={downloadExtract}>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
							<polyline points="7 10 12 15 17 10"/>
							<line x1="12" x2="12" y1="15" y2="3"/>
						</svg>
						Download PDF
					</Button>
					<Button onclick={orderCertifiedExtract}>
						Order Certified
					</Button>
				</div>
			</div>
		</div>
	</div>
{/if}
