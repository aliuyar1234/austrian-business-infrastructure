<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import {
		Building2, MapPin, Users, Calendar, Euro, Plus, Edit2, Trash2,
		Check, AlertCircle, ChevronRight, RefreshCw, Search
	} from 'lucide-svelte';
	import type { Unternehmensprofil, ProfileListResponse } from '$lib/api/foerderung.types';

	// Extended interface for display purposes
	interface DisplayProfile extends Unternehmensprofil {
		status?: string;
		derived_from_account?: boolean;
		last_search_at?: string;
	}

	let profiles: DisplayProfile[] = [];
	let loading = true;
	let error = '';

	async function loadProfiles() {
		loading = true;
		error = '';
		try {
			const response = await api.get<ProfileListResponse>('/profile?limit=100');
			profiles = response.profiles || [];
		} catch (e) {
			error = 'Fehler beim Laden der Profile';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	async function deleteProfile(id: string) {
		if (!confirm('Möchten Sie dieses Profil wirklich löschen?')) return;
		try {
			await api.delete(`/profile/${id}`);
			profiles = profiles.filter(p => p.id !== id);
		} catch (e) {
			error = 'Fehler beim Löschen des Profils';
			console.error(e);
		}
	}

	function formatAmount(amount?: number): string {
		if (!amount) return '-';
		return new Intl.NumberFormat('de-AT', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(amount);
	}

	function getStatusBadge(profile: Profile): { text: string; class: string } {
		if (profile.status === 'complete') {
			return { text: 'Vollständig', class: 'bg-green-100 text-green-800' };
		} else if (profile.status === 'draft') {
			return { text: 'Entwurf', class: 'bg-gray-100 text-gray-800' };
		}
		return { text: profile.status, class: 'bg-gray-100 text-gray-800' };
	}

	onMount(loadProfiles);
</script>

<svelte:head>
	<title>Unternehmensprofile | Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<div class="flex items-center justify-between mb-8">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Unternehmensprofile</h1>
			<p class="text-sm text-gray-500 mt-1">Verwalten Sie Ihre Unternehmensprofile für die Förderungssuche</p>
		</div>
		<a href="/profile/new" class="btn btn-primary flex items-center gap-2">
			<Plus class="w-4 h-4" />
			Neues Profil
		</a>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if error}
		<div class="alert alert-error">{error}</div>
	{:else if profiles.length === 0}
		<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
			<Building2 class="w-12 h-12 mx-auto mb-4 text-gray-400" />
			<h3 class="text-lg font-medium text-gray-900 mb-2">Noch keine Profile</h3>
			<p class="text-gray-600 mb-4">Erstellen Sie ein Unternehmensprofil für die Förderungssuche.</p>
			<a href="/profile/new" class="btn btn-primary">
				<Plus class="w-4 h-4 mr-2" />
				Profil erstellen
			</a>
		</div>
	{:else}
		<div class="grid gap-4">
			{#each profiles as profile}
				{@const status = getStatusBadge(profile)}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 hover:shadow-md transition-shadow">
					<div class="p-6">
						<div class="flex items-start justify-between">
							<div class="flex-1">
								<div class="flex items-center gap-2 mb-2">
									<span class="px-2 py-0.5 text-xs font-medium rounded-full {status.class}">
										{status.text}
									</span>
									{#if profile.derived_from_account}
										<span class="px-2 py-0.5 text-xs bg-blue-100 text-blue-700 rounded-full flex items-center gap-1">
											<RefreshCw class="w-3 h-3" />
											Automatisch
										</span>
									{/if}
									{#if profile.is_kmu}
										<span class="px-2 py-0.5 text-xs bg-purple-100 text-purple-700 rounded-full">
											KMU
										</span>
									{/if}
									{#if profile.is_startup}
										<span class="px-2 py-0.5 text-xs bg-green-100 text-green-700 rounded-full">
											Startup
										</span>
									{/if}
								</div>
								<h3 class="text-lg font-semibold text-gray-900">{profile.name}</h3>
								{#if profile.legal_form}
									<p class="text-gray-500 text-sm">{profile.legal_form}</p>
								{/if}

								<div class="flex flex-wrap items-center gap-4 mt-3 text-sm text-gray-600">
									{#if profile.state}
										<div class="flex items-center gap-1">
											<MapPin class="w-4 h-4" />
											{profile.state}{profile.district ? `, ${profile.district}` : ''}
										</div>
									{/if}
									{#if profile.employees_count}
										<div class="flex items-center gap-1">
											<Users class="w-4 h-4" />
											{profile.employees_count} Mitarbeiter
										</div>
									{/if}
									{#if profile.founded_year}
										<div class="flex items-center gap-1">
											<Calendar class="w-4 h-4" />
											Gegründet {profile.founded_year}
										</div>
									{/if}
									{#if profile.annual_revenue}
										<div class="flex items-center gap-1">
											<Euro class="w-4 h-4" />
											{formatAmount(profile.annual_revenue)} Umsatz
										</div>
									{/if}
								</div>

								{#if profile.project_topics && profile.project_topics.length > 0}
									<div class="flex flex-wrap gap-1 mt-3">
										{#each profile.project_topics.slice(0, 5) as topic}
											<span class="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded">
												{topic}
											</span>
										{/each}
										{#if profile.project_topics.length > 5}
											<span class="text-xs text-gray-500">+{profile.project_topics.length - 5}</span>
										{/if}
									</div>
								{/if}

								{#if profile.last_search_at}
									<p class="text-xs text-gray-400 mt-3">
										Letzte Suche: {new Date(profile.last_search_at).toLocaleDateString('de-AT')}
									</p>
								{/if}
							</div>

							<div class="flex items-center gap-2 ml-4">
								<a href="/foerderungen/suche?profile={profile.id}" class="btn btn-primary btn-sm flex items-center gap-1">
									<Search class="w-4 h-4" />
									Suche
								</a>
								<a href="/profile/{profile.id}" class="btn btn-ghost btn-sm">
									<Edit2 class="w-4 h-4" />
								</a>
								<button on:click={() => deleteProfile(profile.id)} class="btn btn-ghost btn-sm text-red-600 hover:bg-red-50">
									<Trash2 class="w-4 h-4" />
								</button>
							</div>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
