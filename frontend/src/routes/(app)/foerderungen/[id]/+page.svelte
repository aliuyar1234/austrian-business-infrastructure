<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import {
		ArrowLeft, ExternalLink, Euro, Calendar, Building2, Users,
		MapPin, Tag, Check, X, AlertCircle, Plus, FileText
	} from 'lucide-svelte';

	interface Foerderung {
		id: string;
		name: string;
		short_name?: string;
		description?: string;
		provider: string;
		type: string;
		funding_rate_min?: number;
		funding_rate_max?: number;
		max_amount?: number;
		min_amount?: number;
		target_size?: string;
		target_age?: string;
		target_legal_forms?: string[];
		target_industries?: string[];
		target_states?: string[];
		topics?: string[];
		categories?: string[];
		requirements?: string;
		eligibility_criteria?: object;
		application_deadline?: string;
		deadline_type?: string;
		call_start?: string;
		call_end?: string;
		url?: string;
		application_url?: string;
		guideline_url?: string;
		combinable_with?: string[];
		status: string;
		is_highlighted: boolean;
	}

	interface CombinationAnalysis {
		combinable_with: Array<{
			foerderung: Foerderung;
			combination_type: string;
			notes?: string;
		}>;
		warnings?: string[];
	}

	let foerderung: Foerderung | null = null;
	let combinations: CombinationAnalysis | null = null;
	let loading = true;
	let error = '';

	$: foerderungId = $page.params.id;

	async function loadFoerderung() {
		loading = true;
		error = '';
		try {
			foerderung = await api.get<Foerderung>(`/foerderungen/${foerderungId}`);
			// Load combinations
			try {
				combinations = await api.get<CombinationAnalysis>(`/foerderungen/${foerderungId}/combinations`);
			} catch (e) {
				console.error('Failed to load combinations', e);
			}
		} catch (e) {
			error = 'Förderung nicht gefunden';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	function formatAmount(amount?: number): string {
		if (!amount) return '-';
		return new Intl.NumberFormat('de-AT', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(amount);
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

	function getTargetSizeLabel(size?: string): string {
		const labels: Record<string, string> = {
			'all': 'Alle Unternehmensgrößen',
			'startup': 'Startups',
			'mikro': 'Kleinstunternehmen (<10 MA)',
			'klein': 'Kleinunternehmen (<50 MA)',
			'mittel': 'Mittelunternehmen (<250 MA)',
			'kmu': 'KMU (<250 MA)',
			'gross': 'Großunternehmen'
		};
		return labels[size || ''] || size || '-';
	}

	onMount(loadFoerderung);
</script>

<svelte:head>
	<title>{foerderung?.name || 'Förderung'} | Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<a href="/foerderungen" class="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mb-6">
		<ArrowLeft class="w-4 h-4" />
		Zurück zur Übersicht
	</a>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if error}
		<div class="alert alert-error">{error}</div>
	{:else if foerderung}
		<!-- Header -->
		<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-6">
			<div class="flex items-start justify-between mb-4">
				<div>
					<div class="flex items-center gap-2 mb-2">
						<span class="px-2 py-0.5 text-xs font-medium rounded-full {getProviderColor(foerderung.provider)}">
							{foerderung.provider}
						</span>
						<span class="text-xs text-gray-500">{getTypeLabel(foerderung.type)}</span>
						{#if foerderung.status === 'active'}
							<span class="px-2 py-0.5 text-xs bg-green-100 text-green-800 rounded-full">Aktiv</span>
						{:else}
							<span class="px-2 py-0.5 text-xs bg-gray-100 text-gray-800 rounded-full">{foerderung.status}</span>
						{/if}
					</div>
					<h1 class="text-2xl font-bold text-gray-900">{foerderung.name}</h1>
					{#if foerderung.short_name}
						<p class="text-gray-500">{foerderung.short_name}</p>
					{/if}
				</div>
				{#if foerderung.url}
					<a href={foerderung.url} target="_blank" rel="noopener" class="btn btn-outline btn-sm flex items-center gap-2">
						<ExternalLink class="w-4 h-4" />
						Website
					</a>
				{/if}
			</div>

			{#if foerderung.description}
				<p class="text-gray-600 mb-6">{foerderung.description}</p>
			{/if}

			<!-- Key Metrics -->
			<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
				<div class="bg-gray-50 rounded-lg p-4">
					<div class="flex items-center gap-2 text-gray-500 text-sm mb-1">
						<Euro class="w-4 h-4" />
						Maximalbetrag
					</div>
					<div class="text-lg font-semibold text-gray-900">
						{formatAmount(foerderung.max_amount)}
					</div>
				</div>
				<div class="bg-gray-50 rounded-lg p-4">
					<div class="flex items-center gap-2 text-gray-500 text-sm mb-1">
						<Euro class="w-4 h-4" />
						Förderquote
					</div>
					<div class="text-lg font-semibold text-gray-900">
						{#if foerderung.funding_rate_min && foerderung.funding_rate_max}
							{foerderung.funding_rate_min}% - {foerderung.funding_rate_max}%
						{:else if foerderung.funding_rate_max}
							bis {foerderung.funding_rate_max}%
						{:else}
							-
						{/if}
					</div>
				</div>
				<div class="bg-gray-50 rounded-lg p-4">
					<div class="flex items-center gap-2 text-gray-500 text-sm mb-1">
						<Calendar class="w-4 h-4" />
						Einreichfrist
					</div>
					<div class="text-lg font-semibold text-gray-900">
						{#if foerderung.application_deadline}
							{new Date(foerderung.application_deadline).toLocaleDateString('de-AT')}
						{:else if foerderung.deadline_type === 'laufend'}
							Laufend
						{:else}
							-
						{/if}
					</div>
				</div>
				<div class="bg-gray-50 rounded-lg p-4">
					<div class="flex items-center gap-2 text-gray-500 text-sm mb-1">
						<Users class="w-4 h-4" />
						Zielgruppe
					</div>
					<div class="text-lg font-semibold text-gray-900">
						{getTargetSizeLabel(foerderung.target_size)}
					</div>
				</div>
			</div>
		</div>

		<!-- Details -->
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
			<div class="lg:col-span-2 space-y-6">
				<!-- Requirements -->
				{#if foerderung.requirements}
					<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
						<h2 class="text-lg font-semibold text-gray-900 mb-4">Voraussetzungen</h2>
						<div class="prose prose-sm max-w-none text-gray-600">
							{foerderung.requirements}
						</div>
					</div>
				{/if}

				<!-- Topics -->
				{#if foerderung.topics && foerderung.topics.length > 0}
					<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
						<h2 class="text-lg font-semibold text-gray-900 mb-4">Themen</h2>
						<div class="flex flex-wrap gap-2">
							{#each foerderung.topics as topic}
								<span class="px-3 py-1 bg-blue-50 text-blue-700 rounded-full text-sm">{topic}</span>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Kombinierbare Förderungen -->
				{#if combinations && combinations.combinable_with && combinations.combinable_with.length > 0}
					<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
						<h2 class="text-lg font-semibold text-gray-900 mb-4">Kombinierbare Förderungen</h2>
						{#if combinations.warnings && combinations.warnings.length > 0}
							<div class="alert alert-warning mb-4">
								<AlertCircle class="w-4 h-4" />
								<ul class="list-disc list-inside text-sm">
									{#each combinations.warnings as warning}
										<li>{warning}</li>
									{/each}
								</ul>
							</div>
						{/if}
						<div class="space-y-3">
							{#each combinations.combinable_with as combo}
								<a href="/foerderungen/{combo.foerderung.id}" class="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
									<div>
										<div class="font-medium text-gray-900">{combo.foerderung.name}</div>
										<div class="text-sm text-gray-500">
											{combo.foerderung.provider} | {getTypeLabel(combo.foerderung.type)}
											{#if combo.notes}
												<span class="text-green-600"> | {combo.notes}</span>
											{/if}
										</div>
									</div>
									<span class="text-xs px-2 py-0.5 bg-blue-100 text-blue-700 rounded">
										{combo.combination_type === 'explicit' ? 'Bestätigt' : 'Möglich'}
									</span>
								</a>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<!-- Sidebar -->
			<div class="space-y-6">
				<!-- Eligible Regions -->
				{#if foerderung.target_states && foerderung.target_states.length > 0}
					<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
						<h3 class="font-semibold text-gray-900 mb-3">Bundesländer</h3>
						<div class="flex flex-wrap gap-2">
							{#each foerderung.target_states as state}
								<span class="px-2 py-1 bg-gray-100 text-gray-700 rounded text-sm">{state}</span>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Legal Forms -->
				{#if foerderung.target_legal_forms && foerderung.target_legal_forms.length > 0}
					<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
						<h3 class="font-semibold text-gray-900 mb-3">Rechtsformen</h3>
						<div class="flex flex-wrap gap-2">
							{#each foerderung.target_legal_forms as form}
								<span class="px-2 py-1 bg-gray-100 text-gray-700 rounded text-sm">{form}</span>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Links -->
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
					<h3 class="font-semibold text-gray-900 mb-3">Links</h3>
					<div class="space-y-2">
						{#if foerderung.application_url}
							<a href={foerderung.application_url} target="_blank" rel="noopener" class="flex items-center gap-2 text-blue-600 hover:underline">
								<Plus class="w-4 h-4" />
								Antrag stellen
							</a>
						{/if}
						{#if foerderung.guideline_url}
							<a href={foerderung.guideline_url} target="_blank" rel="noopener" class="flex items-center gap-2 text-blue-600 hover:underline">
								<FileText class="w-4 h-4" />
								Förderrichtlinie
							</a>
						{/if}
						{#if foerderung.url}
							<a href={foerderung.url} target="_blank" rel="noopener" class="flex items-center gap-2 text-blue-600 hover:underline">
								<ExternalLink class="w-4 h-4" />
								Förderstelle
							</a>
						{/if}
					</div>
				</div>

				<!-- Actions -->
				<div class="bg-blue-50 rounded-lg p-6">
					<h3 class="font-semibold text-gray-900 mb-3">Passende Förderung?</h3>
					<p class="text-sm text-gray-600 mb-4">
						Starten Sie die Förderungssuche um zu prüfen, ob diese Förderung zu Ihrem Unternehmen passt.
					</p>
					<a href="/foerderungen/suche" class="btn btn-primary w-full">
						Förderungssuche starten
					</a>
				</div>
			</div>
		</div>
	{/if}
</div>
