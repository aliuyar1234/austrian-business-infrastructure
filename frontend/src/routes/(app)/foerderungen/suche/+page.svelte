<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import {
		Search, Building2, MapPin, Users, Tag, Briefcase, Euro,
		ChevronRight, Sparkles, Check, AlertCircle, Download, FileText, Clock
	} from 'lucide-svelte';
	import type { Unternehmensprofil, ProfileListResponse, SearchResponse, FoerderungsMatch } from '$lib/api/foerderung.types';

	// Local display interface extending the base match type
	interface DisplayMatchResult extends FoerderungsMatch {
		type?: string;
		max_amount?: number;
		matched_criteria?: string[];
		concerns?: string[];
		insider_tipp?: string;
	}

	interface DisplaySearchResult {
		search_id: string;
		matches: DisplayMatchResult[];
		llm_tokens_used: number;
		duration_ms?: number;
		duration?: number;
		llm_fallback: boolean;
	}

	// Form state
	let step = 1;
	let selectedProfileId = '';
	let companyName = '';
	let state = '';
	let employeesCount: number | null = null;
	let isStartup = false;
	let projectDescription = '';
	let selectedTopics: string[] = [];

	// Results
	let profiles: Unternehmensprofil[] = [];
	let searchResult: DisplaySearchResult | null = null;
	let loading = false;
	let searching = false;
	let error = '';

	const states = [
		'Wien', 'Steiermark', 'Oberösterreich', 'Niederösterreich',
		'Salzburg', 'Tirol', 'Kärnten', 'Vorarlberg', 'Burgenland'
	];

	const topics = [
		'Innovation', 'Digitalisierung', 'Forschung & Entwicklung', 'Energie & Umwelt',
		'Export & Internationalisierung', 'Gründung', 'Wachstum', 'Beschäftigung',
		'Investitionen', 'Nachhaltigkeit', 'Kreislaufwirtschaft', 'KI & Automatisierung'
	];

	async function loadProfiles() {
		try {
			const response = await api.get<ProfileListResponse>('/api/v1/profile?limit=100');
			profiles = response.profiles || [];
		} catch (e) {
			console.error('Failed to load profiles', e);
		}
	}

	function selectProfile(profile: Unternehmensprofil) {
		selectedProfileId = profile.id;
		companyName = profile.name;
		state = profile.state || '';
		employeesCount = profile.employees_count || null;
		isStartup = profile.is_startup;
		step = 2;
	}

	function toggleTopic(topic: string) {
		if (selectedTopics.includes(topic)) {
			selectedTopics = selectedTopics.filter(t => t !== topic);
		} else {
			selectedTopics = [...selectedTopics, topic];
		}
	}

	async function runSearch() {
		if (!companyName || !state || selectedTopics.length === 0) {
			error = 'Bitte füllen Sie alle Pflichtfelder aus';
			return;
		}

		searching = true;
		error = '';
		try {
			const request = selectedProfileId
				? { profile_id: selectedProfileId }
				: {
					company_name: companyName,
					state: state,
					employees_count: employeesCount,
					is_startup: isStartup,
					project_topics: selectedTopics,
					project_description: projectDescription
				};

			searchResult = await api.post('/api/v1/foerderungssuche', request);
			step = 3;
		} catch (e) {
			error = 'Fehler bei der Suche. Bitte versuchen Sie es erneut.';
			console.error(e);
		} finally {
			searching = false;
		}
	}

	function formatAmount(amount?: number): string {
		if (!amount) return '-';
		if (amount >= 1000000) return `€${(amount / 1000000).toFixed(1)}M`;
		if (amount >= 1000) return `€${(amount / 1000).toFixed(0)}k`;
		return `€${amount}`;
	}

	function getScoreColor(score: number): string {
		if (score >= 80) return 'text-green-600';
		if (score >= 60) return 'text-amber-600';
		return 'text-gray-600';
	}

	function getScoreBg(score: number): string {
		if (score >= 80) return 'bg-green-500';
		if (score >= 60) return 'bg-amber-500';
		return 'bg-gray-400';
	}

	async function exportResults(format: 'markdown' | 'pdf') {
		if (!searchResult) return;
		try {
			const response = await fetch(`/api/v1/foerderungssuche/${searchResult.search_id}/export?format=${format}`, {
				headers: {
					'Accept': format === 'pdf' ? 'application/pdf' : 'text/markdown'
				}
			});
			const blob = await response.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `foerderungen-${new Date().toISOString().split('T')[0]}.${format === 'pdf' ? 'pdf' : 'md'}`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			console.error('Export failed', e);
		}
	}

	function startOver() {
		step = 1;
		searchResult = null;
		selectedProfileId = '';
		companyName = '';
		state = '';
		employeesCount = null;
		isStartup = false;
		projectDescription = '';
		selectedTopics = [];
	}

	onMount(loadProfiles);
</script>

<svelte:head>
	<title>Förderungssuche | Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<!-- Progress Steps -->
	<div class="flex items-center justify-center mb-8">
		<div class="flex items-center gap-4">
			<div class="flex items-center gap-2 {step >= 1 ? 'text-blue-600' : 'text-gray-400'}">
				<div class="w-8 h-8 rounded-full {step >= 1 ? 'bg-blue-600 text-white' : 'bg-gray-200'} flex items-center justify-center font-medium">1</div>
				<span class="hidden sm:inline">Profil</span>
			</div>
			<div class="w-12 h-0.5 {step >= 2 ? 'bg-blue-600' : 'bg-gray-200'}"></div>
			<div class="flex items-center gap-2 {step >= 2 ? 'text-blue-600' : 'text-gray-400'}">
				<div class="w-8 h-8 rounded-full {step >= 2 ? 'bg-blue-600 text-white' : 'bg-gray-200'} flex items-center justify-center font-medium">2</div>
				<span class="hidden sm:inline">Vorhaben</span>
			</div>
			<div class="w-12 h-0.5 {step >= 3 ? 'bg-blue-600' : 'bg-gray-200'}"></div>
			<div class="flex items-center gap-2 {step >= 3 ? 'text-blue-600' : 'text-gray-400'}">
				<div class="w-8 h-8 rounded-full {step >= 3 ? 'bg-blue-600 text-white' : 'bg-gray-200'} flex items-center justify-center font-medium">3</div>
				<span class="hidden sm:inline">Ergebnisse</span>
			</div>
		</div>
	</div>

	{#if error}
		<div class="alert alert-error mb-6">{error}</div>
	{/if}

	<!-- Step 1: Profile Selection -->
	{#if step === 1}
		<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
			<h2 class="text-xl font-bold text-gray-900 mb-2">Unternehmensprofil</h2>
			<p class="text-gray-600 mb-6">Wählen Sie ein bestehendes Profil oder geben Sie die Daten manuell ein.</p>

			{#if profiles.length > 0}
				<div class="mb-6">
					<h3 class="font-medium text-gray-900 mb-3">Gespeicherte Profile</h3>
					<div class="grid gap-3">
						{#each profiles as profile}
							<button
								on:click={() => selectProfile(profile)}
								class="flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors text-left"
							>
								<div>
									<div class="font-medium text-gray-900">{profile.name}</div>
									<div class="text-sm text-gray-500">
										{profile.state || 'Kein Bundesland'} | {profile.employees_count || '?'} Mitarbeiter
										{#if profile.is_startup}
											<span class="ml-1 text-green-600">(Startup)</span>
										{/if}
									</div>
								</div>
								<ChevronRight class="w-5 h-5 text-gray-400" />
							</button>
						{/each}
					</div>
				</div>
				<div class="relative my-6">
					<div class="absolute inset-0 flex items-center">
						<div class="w-full border-t border-gray-200"></div>
					</div>
					<div class="relative flex justify-center">
						<span class="bg-white px-4 text-sm text-gray-500">oder</span>
					</div>
				</div>
			{/if}

			<h3 class="font-medium text-gray-900 mb-3">Manuelle Eingabe</h3>
			<div class="space-y-4">
				<div>
					<label class="label">Unternehmensname *</label>
					<input type="text" bind:value={companyName} class="input input-bordered w-full" placeholder="z.B. Muster GmbH" />
				</div>
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label class="label">Bundesland *</label>
						<select bind:value={state} class="select select-bordered w-full">
							<option value="">Bitte wählen</option>
							{#each states as s}
								<option value={s}>{s}</option>
							{/each}
						</select>
					</div>
					<div>
						<label class="label">Mitarbeiteranzahl</label>
						<input type="number" bind:value={employeesCount} class="input input-bordered w-full" placeholder="z.B. 15" />
					</div>
				</div>
				<div class="flex items-center gap-2">
					<input type="checkbox" bind:checked={isStartup} class="checkbox checkbox-primary" />
					<label class="text-gray-700">Startup (bis 5 Jahre alt)</label>
				</div>
			</div>

			<div class="flex justify-end mt-6">
				<button
					on:click={() => { if (companyName && state) step = 2; else error = 'Bitte Unternehmensname und Bundesland angeben'; }}
					class="btn btn-primary"
				>
					Weiter
				</button>
			</div>
		</div>
	{/if}

	<!-- Step 2: Project Description -->
	{#if step === 2}
		<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
			<h2 class="text-xl font-bold text-gray-900 mb-2">Ihr Vorhaben</h2>
			<p class="text-gray-600 mb-6">Beschreiben Sie Ihr geplantes Projekt und wählen Sie relevante Themen.</p>

			<div class="space-y-6">
				<div>
					<label class="label">Themen *</label>
					<p class="text-sm text-gray-500 mb-3">Wählen Sie mindestens ein Thema, das zu Ihrem Vorhaben passt.</p>
					<div class="flex flex-wrap gap-2">
						{#each topics as topic}
							<button
								on:click={() => toggleTopic(topic)}
								class="px-3 py-1.5 rounded-full text-sm border transition-colors {selectedTopics.includes(topic) ? 'bg-blue-600 text-white border-blue-600' : 'bg-white text-gray-700 border-gray-300 hover:border-blue-500'}"
							>
								{#if selectedTopics.includes(topic)}
									<Check class="w-3 h-3 inline mr-1" />
								{/if}
								{topic}
							</button>
						{/each}
					</div>
				</div>

				<div>
					<label class="label">Projektbeschreibung (optional)</label>
					<textarea
						bind:value={projectDescription}
						class="textarea textarea-bordered w-full h-32"
						placeholder="Beschreiben Sie kurz Ihr geplantes Vorhaben..."
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Eine detaillierte Beschreibung hilft bei der KI-gestützten Analyse</p>
				</div>
			</div>

			<div class="flex justify-between mt-6">
				<button on:click={() => step = 1} class="btn btn-ghost">Zurück</button>
				<button on:click={runSearch} disabled={searching || selectedTopics.length === 0} class="btn btn-primary">
					{#if searching}
						<span class="loading loading-spinner loading-sm"></span>
						Analysiere...
					{:else}
						<Sparkles class="w-4 h-4 mr-2" />
						Förderungen finden
					{/if}
				</button>
			</div>
		</div>
	{/if}

	<!-- Step 3: Results -->
	{#if step === 3 && searchResult}
		<div class="space-y-6">
			<!-- Summary -->
			<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
				<div class="flex items-center justify-between mb-4">
					<div>
						<h2 class="text-xl font-bold text-gray-900">Suchergebnisse</h2>
						<p class="text-gray-600">
							{searchResult.matches.length} passende Förderungen gefunden
							<span class="text-sm text-gray-400">
								({(searchResult.duration_ms / 1000).toFixed(1)}s)
							</span>
						</p>
					</div>
					<div class="flex gap-2">
						<button on:click={() => exportResults('markdown')} class="btn btn-outline btn-sm">
							<FileText class="w-4 h-4 mr-1" />
							Markdown
						</button>
						<button on:click={() => exportResults('pdf')} class="btn btn-outline btn-sm">
							<Download class="w-4 h-4 mr-1" />
							PDF
						</button>
					</div>
				</div>

				{#if searchResult.llm_fallback}
					<div class="alert alert-warning mb-4">
						<AlertCircle class="w-4 h-4" />
						<span>KI-Analyse war nicht verfügbar. Ergebnisse basieren nur auf Regelwerk.</span>
					</div>
				{/if}
			</div>

			<!-- Results List -->
			{#each searchResult.matches as match, index}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
					<div class="p-6">
						<div class="flex items-start justify-between">
							<div class="flex-1">
								<div class="flex items-center gap-2 mb-2">
									<span class="text-sm font-bold text-gray-400">#{index + 1}</span>
									<span class="px-2 py-0.5 text-xs bg-gray-100 text-gray-700 rounded-full">
										{match.provider}
									</span>
								</div>
								<h3 class="text-lg font-semibold text-gray-900">{match.foerderung_name}</h3>

								<!-- Match Score -->
								<div class="flex items-center gap-4 mt-3">
									<div class="flex items-center gap-2">
										<div class="w-24 h-2 bg-gray-200 rounded-full overflow-hidden">
											<div class="h-full {getScoreBg(match.total_score)}" style="width: {match.total_score}%"></div>
										</div>
										<span class="{getScoreColor(match.total_score)} font-semibold">{match.total_score}%</span>
									</div>
									{#if match.max_amount}
										<span class="text-sm text-gray-600">
											<Euro class="w-4 h-4 inline" />
											bis {formatAmount(match.max_amount)}
										</span>
									{/if}
								</div>

								<!-- Matched Criteria -->
								{#if match.matched_criteria && match.matched_criteria.length > 0}
									<div class="mt-4">
										<div class="text-sm font-medium text-gray-700 mb-2">Passende Kriterien:</div>
										<div class="flex flex-wrap gap-2">
											{#each match.matched_criteria as criteria}
												<span class="inline-flex items-center gap-1 px-2 py-1 bg-green-50 text-green-700 text-sm rounded">
													<Check class="w-3 h-3" />
													{criteria}
												</span>
											{/each}
										</div>
									</div>
								{/if}

								<!-- Concerns -->
								{#if match.concerns && match.concerns.length > 0}
									<div class="mt-3">
										<div class="text-sm font-medium text-gray-700 mb-2">Zu beachten:</div>
										<ul class="text-sm text-amber-700 space-y-1">
											{#each match.concerns as concern}
												<li class="flex items-start gap-2">
													<AlertCircle class="w-4 h-4 flex-shrink-0 mt-0.5" />
													{concern}
												</li>
											{/each}
										</ul>
									</div>
								{/if}

								<!-- Insider Tip -->
								{#if match.insider_tipp}
									<div class="mt-3 p-3 bg-blue-50 rounded-lg">
										<div class="flex items-center gap-2 text-blue-700 text-sm font-medium mb-1">
											<Sparkles class="w-4 h-4" />
											Insider-Tipp
										</div>
										<p class="text-sm text-blue-800">{match.insider_tipp}</p>
									</div>
								{/if}
							</div>

							<a href="/foerderungen/{match.foerderung_id}" class="btn btn-outline btn-sm flex-shrink-0 ml-4">
								Details
								<ChevronRight class="w-4 h-4" />
							</a>
						</div>
					</div>
				</div>
			{/each}

			{#if searchResult.matches.length === 0}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<Building2 class="w-12 h-12 mx-auto mb-4 text-gray-400" />
					<h3 class="text-lg font-medium text-gray-900 mb-2">Keine passenden Förderungen gefunden</h3>
					<p class="text-gray-600 mb-4">Versuchen Sie es mit anderen Themen oder Kriterien.</p>
					<button on:click={startOver} class="btn btn-primary">Neue Suche starten</button>
				</div>
			{:else}
				<div class="text-center">
					<button on:click={startOver} class="btn btn-outline">
						Neue Suche starten
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>
