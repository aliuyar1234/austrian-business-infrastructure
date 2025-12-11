<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { Search, Filter, ExternalLink, Euro, Users, Building2, ChevronRight, Star } from 'lucide-svelte';
	import type { Foerderung, FoerderungenListResponse } from '$lib/api/foerderung.types';

	// Extended local interface for display purposes
	interface DisplayFoerderung extends Foerderung {
		short_name?: string;
		is_highlighted?: boolean;
	}

	let foerderungen: DisplayFoerderung[] = [];
	let loading = true;
	let error = '';
	let total = 0;
	let page = 1;
	let limit = 20;

	// Filters
	let searchQuery = '';
	let selectedProvider = '';
	let selectedType = '';
	let selectedState = '';
	let selectedTopic = '';

	const providers = ['AWS', 'FFG', 'WKO', 'AMS', 'OeKB', 'EU', 'SFG', 'WIBAG', 'WIST', 'KWF'];
	const types = [
		{ value: 'zuschuss', label: 'Zuschuss' },
		{ value: 'kredit', label: 'Kredit' },
		{ value: 'garantie', label: 'Garantie' },
		{ value: 'beteiligung', label: 'Beteiligung' },
		{ value: 'haftung', label: 'Haftung' }
	];
	const states = ['Wien', 'Steiermark', 'Oberösterreich', 'Niederösterreich', 'Salzburg', 'Tirol', 'Kärnten', 'Vorarlberg', 'Burgenland'];
	const topics = ['Innovation', 'Digitalisierung', 'Energie', 'Umwelt', 'Export', 'Forschung', 'Gründung', 'Wachstum'];

	async function loadFoerderungen() {
		loading = true;
		error = '';
		try {
			const params = new URLSearchParams();
			params.set('limit', limit.toString());
			params.set('offset', ((page - 1) * limit).toString());
			if (searchQuery) params.set('search', searchQuery);
			if (selectedProvider) params.set('provider', selectedProvider);
			if (selectedType) params.set('type', selectedType);
			if (selectedState) params.set('state', selectedState);
			if (selectedTopic) params.set('topic', selectedTopic);
			params.set('status', 'active');

			const response = await api.get<FoerderungenListResponse>(`/api/v1/foerderungen?${params}`);
			foerderungen = response.foerderungen || [];
			total = response.total || 0;
		} catch (e) {
			error = 'Fehler beim Laden der Förderungen';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	function handleSearch() {
		page = 1;
		loadFoerderungen();
	}

	function clearFilters() {
		searchQuery = '';
		selectedProvider = '';
		selectedType = '';
		selectedState = '';
		selectedTopic = '';
		page = 1;
		loadFoerderungen();
	}

	function formatAmount(amount?: number): string {
		if (!amount) return '-';
		if (amount >= 1000000) return `€${(amount / 1000000).toFixed(1)}M`;
		if (amount >= 1000) return `€${(amount / 1000).toFixed(0)}k`;
		return `€${amount}`;
	}

	function getProviderColor(provider: string): string {
		const colors: Record<string, string> = {
			'AWS': 'bg-blue-100 text-blue-800',
			'FFG': 'bg-green-100 text-green-800',
			'WKO': 'bg-amber-100 text-amber-800',
			'AMS': 'bg-purple-100 text-purple-800',
			'OeKB': 'bg-rose-100 text-rose-800',
			'EU': 'bg-indigo-100 text-indigo-800'
		};
		return colors[provider] || 'bg-gray-100 text-gray-800';
	}

	function getTypeLabel(type: string): string {
		const labels: Record<string, string> = {
			'zuschuss': 'Zuschuss',
			'kredit': 'Kredit',
			'garantie': 'Garantie',
			'beteiligung': 'Beteiligung',
			'haftung': 'Haftung'
		};
		return labels[type] || type;
	}

	onMount(loadFoerderungen);
</script>

<svelte:head>
	<title>Förderungen | Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<div class="flex items-center justify-between mb-8">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Förderungsdatenbank</h1>
			<p class="text-sm text-gray-500 mt-1">{total} aktive Förderungsprogramme</p>
		</div>
		<a href="/foerderungen/suche" class="btn btn-primary flex items-center gap-2">
			<Search class="w-4 h-4" />
			Förderungssuche starten
		</a>
	</div>

	<!-- Filters -->
	<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4 mb-6">
		<div class="flex items-center gap-2 mb-4">
			<Filter class="w-5 h-5 text-gray-400" />
			<span class="font-medium text-gray-700">Filter</span>
		</div>
		<div class="grid grid-cols-1 md:grid-cols-5 gap-4">
			<div>
				<input
					type="text"
					placeholder="Suche..."
					bind:value={searchQuery}
					on:keydown={(e) => e.key === 'Enter' && handleSearch()}
					class="input input-bordered w-full"
				/>
			</div>
			<div>
				<select bind:value={selectedProvider} on:change={handleSearch} class="select select-bordered w-full">
					<option value="">Alle Fördergeber</option>
					{#each providers as provider}
						<option value={provider}>{provider}</option>
					{/each}
				</select>
			</div>
			<div>
				<select bind:value={selectedType} on:change={handleSearch} class="select select-bordered w-full">
					<option value="">Alle Arten</option>
					{#each types as type}
						<option value={type.value}>{type.label}</option>
					{/each}
				</select>
			</div>
			<div>
				<select bind:value={selectedState} on:change={handleSearch} class="select select-bordered w-full">
					<option value="">Alle Bundesländer</option>
					{#each states as state}
						<option value={state}>{state}</option>
					{/each}
				</select>
			</div>
			<div class="flex gap-2">
				<button on:click={handleSearch} class="btn btn-primary flex-1">Suchen</button>
				<button on:click={clearFilters} class="btn btn-ghost">Zurücksetzen</button>
			</div>
		</div>
	</div>

	<!-- Results -->
	{#if loading}
		<div class="flex items-center justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if error}
		<div class="alert alert-error">{error}</div>
	{:else if foerderungen.length === 0}
		<div class="text-center py-12 text-gray-500">
			<Building2 class="w-12 h-12 mx-auto mb-4 text-gray-400" />
			<p>Keine Förderungen gefunden</p>
		</div>
	{:else}
		<div class="space-y-4">
			{#each foerderungen as f}
				<a href="/foerderungen/{f.id}" class="block bg-white rounded-lg shadow-sm border border-gray-200 hover:shadow-md transition-shadow">
					<div class="p-6">
						<div class="flex items-start justify-between">
							<div class="flex-1">
								<div class="flex items-center gap-2 mb-2">
									{#if f.is_highlighted}
										<Star class="w-4 h-4 text-amber-500 fill-amber-500" />
									{/if}
									<span class="px-2 py-0.5 text-xs font-medium rounded-full {getProviderColor(f.provider)}">
										{f.provider}
									</span>
									<span class="text-xs text-gray-500">{getTypeLabel(f.type)}</span>
								</div>
								<h3 class="text-lg font-semibold text-gray-900">{f.name}</h3>
								{#if f.description}
									<p class="text-sm text-gray-600 mt-1 line-clamp-2">{f.description}</p>
								{/if}
								<div class="flex flex-wrap items-center gap-4 mt-3">
									{#if f.max_amount}
										<div class="flex items-center gap-1 text-sm text-gray-600">
											<Euro class="w-4 h-4" />
											<span>bis {formatAmount(f.max_amount)}</span>
										</div>
									{/if}
									{#if f.funding_rate_max}
										<div class="text-sm text-gray-600">
											bis {f.funding_rate_max}% Förderquote
										</div>
									{/if}
									{#if f.application_deadline}
										<div class="text-sm text-orange-600">
											Frist: {new Date(f.application_deadline).toLocaleDateString('de-AT')}
										</div>
									{/if}
								</div>
								{#if f.topics && f.topics.length > 0}
									<div class="flex flex-wrap gap-1 mt-3">
										{#each f.topics.slice(0, 5) as topic}
											<span class="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded">
												{topic}
											</span>
										{/each}
										{#if f.topics.length > 5}
											<span class="text-xs text-gray-500">+{f.topics.length - 5}</span>
										{/if}
									</div>
								{/if}
							</div>
							<ChevronRight class="w-5 h-5 text-gray-400 flex-shrink-0 ml-4" />
						</div>
					</div>
				</a>
			{/each}
		</div>

		<!-- Pagination -->
		{#if total > limit}
			<div class="flex items-center justify-center gap-2 mt-8">
				<button
					on:click={() => { page--; loadFoerderungen(); }}
					disabled={page === 1}
					class="btn btn-sm btn-ghost"
				>
					Zurück
				</button>
				<span class="text-sm text-gray-600">
					Seite {page} von {Math.ceil(total / limit)}
				</span>
				<button
					on:click={() => { page++; loadFoerderungen(); }}
					disabled={page >= Math.ceil(total / limit)}
					class="btn btn-sm btn-ghost"
				>
					Weiter
				</button>
			</div>
		{/if}
	{/if}
</div>

<style>
	.line-clamp-2 {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
