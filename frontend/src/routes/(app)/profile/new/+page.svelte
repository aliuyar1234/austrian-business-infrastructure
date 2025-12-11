<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import { ArrowLeft, Building2, Save, RefreshCw } from 'lucide-svelte';

	let loading = false;
	let deriving = false;
	let error = '';

	// Form fields
	let name = '';
	let legalForm = '';
	let foundedYear: number | null = null;
	let state = '';
	let district = '';
	let employeesCount: number | null = null;
	let annualRevenue: number | null = null;
	let balanceTotal: number | null = null;
	let industry = '';
	let isStartup = false;
	let projectDescription = '';
	let selectedTopics: string[] = [];

	const states = [
		'Wien', 'Steiermark', 'Oberösterreich', 'Niederösterreich',
		'Salzburg', 'Tirol', 'Kärnten', 'Vorarlberg', 'Burgenland'
	];

	const legalForms = [
		'GmbH', 'AG', 'OG', 'KG', 'e.U.', 'Verein', 'Genossenschaft', 'Stiftung', 'Sonstige'
	];

	const topics = [
		'Innovation', 'Digitalisierung', 'Forschung & Entwicklung', 'Energie & Umwelt',
		'Export & Internationalisierung', 'Gründung', 'Wachstum', 'Beschäftigung',
		'Investitionen', 'Nachhaltigkeit', 'Kreislaufwirtschaft', 'KI & Automatisierung'
	];

	function toggleTopic(topic: string) {
		if (selectedTopics.includes(topic)) {
			selectedTopics = selectedTopics.filter(t => t !== topic);
		} else {
			selectedTopics = [...selectedTopics, topic];
		}
	}

	async function handleSubmit() {
		if (!name) {
			error = 'Unternehmensname ist erforderlich';
			return;
		}

		loading = true;
		error = '';
		try {
			const profile = await api.post('/api/v1/profile', {
				name,
				legal_form: legalForm || undefined,
				founded_year: foundedYear || undefined,
				state: state || undefined,
				district: district || undefined,
				employees_count: employeesCount || undefined,
				annual_revenue: annualRevenue || undefined,
				balance_total: balanceTotal || undefined,
				industry: industry || undefined,
				is_startup: isStartup,
				project_description: projectDescription || undefined,
				project_topics: selectedTopics.length > 0 ? selectedTopics : undefined
			});
			goto('/profile');
		} catch (e) {
			error = 'Fehler beim Erstellen des Profils';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	async function deriveFromAccount() {
		// This would derive from a linked account
		deriving = true;
		try {
			// Would call /profile/derive/{accountId}
			await new Promise(resolve => setTimeout(resolve, 1000));
			error = 'Automatische Ableitung erfordert ein verknüpftes Konto';
		} catch (e) {
			error = 'Fehler bei der automatischen Ableitung';
		} finally {
			deriving = false;
		}
	}
</script>

<svelte:head>
	<title>Neues Profil | Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<a href="/profile" class="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mb-6">
		<ArrowLeft class="w-4 h-4" />
		Zurück zur Übersicht
	</a>

	<div class="bg-white rounded-lg shadow-sm border border-gray-200">
		<div class="p-6 border-b border-gray-200">
			<div class="flex items-center justify-between">
				<div>
					<h1 class="text-xl font-bold text-gray-900">Neues Unternehmensprofil</h1>
					<p class="text-sm text-gray-500 mt-1">Erstellen Sie ein Profil für die Förderungssuche</p>
				</div>
				<button on:click={deriveFromAccount} disabled={deriving} class="btn btn-outline btn-sm flex items-center gap-2">
					{#if deriving}
						<span class="loading loading-spinner loading-xs"></span>
					{:else}
						<RefreshCw class="w-4 h-4" />
					{/if}
					Aus Konto ableiten
				</button>
			</div>
		</div>

		{#if error}
			<div class="p-4 border-b border-gray-200">
				<div class="alert alert-error">{error}</div>
			</div>
		{/if}

		<form on:submit|preventDefault={handleSubmit} class="p-6 space-y-6">
			<!-- Basic Info -->
			<div>
				<h2 class="text-lg font-semibold text-gray-900 mb-4">Stammdaten</h2>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div class="md:col-span-2">
						<label class="label">Unternehmensname *</label>
						<input type="text" bind:value={name} class="input input-bordered w-full" placeholder="z.B. Muster GmbH" />
					</div>
					<div>
						<label class="label">Rechtsform</label>
						<select bind:value={legalForm} class="select select-bordered w-full">
							<option value="">Bitte wählen</option>
							{#each legalForms as form}
								<option value={form}>{form}</option>
							{/each}
						</select>
					</div>
					<div>
						<label class="label">Gründungsjahr</label>
						<input type="number" bind:value={foundedYear} class="input input-bordered w-full" min="1900" max={new Date().getFullYear()} placeholder="z.B. 2020" />
					</div>
				</div>
			</div>

			<!-- Location -->
			<div>
				<h2 class="text-lg font-semibold text-gray-900 mb-4">Standort</h2>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="label">Bundesland</label>
						<select bind:value={state} class="select select-bordered w-full">
							<option value="">Bitte wählen</option>
							{#each states as s}
								<option value={s}>{s}</option>
							{/each}
						</select>
					</div>
					<div>
						<label class="label">Bezirk</label>
						<input type="text" bind:value={district} class="input input-bordered w-full" placeholder="z.B. Graz-Stadt" />
					</div>
				</div>
			</div>

			<!-- Size -->
			<div>
				<h2 class="text-lg font-semibold text-gray-900 mb-4">Unternehmensgröße</h2>
				<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
					<div>
						<label class="label">Mitarbeiteranzahl</label>
						<input type="number" bind:value={employeesCount} class="input input-bordered w-full" min="0" placeholder="z.B. 25" />
					</div>
					<div>
						<label class="label">Jahresumsatz (€)</label>
						<input type="number" bind:value={annualRevenue} class="input input-bordered w-full" min="0" placeholder="z.B. 5000000" />
					</div>
					<div>
						<label class="label">Bilanzsumme (€)</label>
						<input type="number" bind:value={balanceTotal} class="input input-bordered w-full" min="0" placeholder="z.B. 3000000" />
					</div>
				</div>
				<div class="mt-4">
					<label class="flex items-center gap-2 cursor-pointer">
						<input type="checkbox" bind:checked={isStartup} class="checkbox checkbox-primary" />
						<span class="text-gray-700">Startup (bis 5 Jahre alt)</span>
					</label>
				</div>
			</div>

			<!-- Industry -->
			<div>
				<h2 class="text-lg font-semibold text-gray-900 mb-4">Branche</h2>
				<input type="text" bind:value={industry} class="input input-bordered w-full" placeholder="z.B. IT & Software, Produktion, Handel" />
			</div>

			<!-- Project -->
			<div>
				<h2 class="text-lg font-semibold text-gray-900 mb-4">Förderprojekt</h2>
				<div class="space-y-4">
					<div>
						<label class="label">Themen</label>
						<div class="flex flex-wrap gap-2">
							{#each topics as topic}
								<button
									type="button"
									on:click={() => toggleTopic(topic)}
									class="px-3 py-1.5 rounded-full text-sm border transition-colors {selectedTopics.includes(topic) ? 'bg-blue-600 text-white border-blue-600' : 'bg-white text-gray-700 border-gray-300 hover:border-blue-500'}"
								>
									{topic}
								</button>
							{/each}
						</div>
					</div>
					<div>
						<label class="label">Projektbeschreibung</label>
						<textarea bind:value={projectDescription} class="textarea textarea-bordered w-full h-32" placeholder="Beschreiben Sie Ihr geplantes Vorhaben..."></textarea>
					</div>
				</div>
			</div>

			<!-- Submit -->
			<div class="flex justify-end gap-4 pt-4 border-t border-gray-200">
				<a href="/profile" class="btn btn-ghost">Abbrechen</a>
				<button type="submit" disabled={loading} class="btn btn-primary">
					{#if loading}
						<span class="loading loading-spinner loading-sm"></span>
					{:else}
						<Save class="w-4 h-4 mr-2" />
					{/if}
					Profil erstellen
				</button>
			</div>
		</form>
	</div>
</div>
