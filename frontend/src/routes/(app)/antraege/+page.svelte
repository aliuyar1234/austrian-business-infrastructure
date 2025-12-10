<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import {
		FileText, Plus, Clock, CheckCircle, XCircle, AlertCircle,
		Filter, ChevronRight, Euro, Calendar, TrendingUp, Building2
	} from 'lucide-svelte';
	import type { Antrag, AntragStats, AntraegeListResponse } from '$lib/api/foerderung.types';

	let antraege: Antrag[] = [];
	let stats: AntragStats | null = null;
	let loading = true;
	let error = '';
	let statusFilter = '';

	const statusLabels: Record<string, { label: string; color: string; icon: typeof Clock }> = {
		'planned': { label: 'Geplant', color: 'bg-gray-100 text-gray-700', icon: Clock },
		'drafting': { label: 'In Bearbeitung', color: 'bg-blue-100 text-blue-700', icon: FileText },
		'submitted': { label: 'Eingereicht', color: 'bg-amber-100 text-amber-700', icon: Clock },
		'in_review': { label: 'In Prüfung', color: 'bg-purple-100 text-purple-700', icon: AlertCircle },
		'approved': { label: 'Bewilligt', color: 'bg-green-100 text-green-700', icon: CheckCircle },
		'rejected': { label: 'Abgelehnt', color: 'bg-red-100 text-red-700', icon: XCircle },
		'withdrawn': { label: 'Zurückgezogen', color: 'bg-gray-100 text-gray-500', icon: XCircle }
	};

	async function loadData() {
		loading = true;
		error = '';
		try {
			const params = new URLSearchParams();
			if (statusFilter) params.set('status', statusFilter);
			params.set('limit', '50');

			const [antraegeResponse, statsResponse] = await Promise.all([
				api.get<AntraegeListResponse>(`/antraege?${params}`),
				api.get<AntragStats>('/antraege/stats')
			]);

			antraege = antraegeResponse.antraege || [];
			stats = statsResponse;
		} catch (e) {
			error = 'Fehler beim Laden der Anträge';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	function formatAmount(amount?: number): string {
		if (!amount) return '-';
		return new Intl.NumberFormat('de-AT', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(amount);
	}

	onMount(loadData);
</script>

<svelte:head>
	<title>Förderanträge | Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<div class="flex items-center justify-between mb-8">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Förderanträge</h1>
			<p class="text-sm text-gray-500 mt-1">Verwalten und verfolgen Sie Ihre Förderanträge</p>
		</div>
		<a href="/antraege/new" class="btn btn-primary flex items-center gap-2">
			<Plus class="w-4 h-4" />
			Neuer Antrag
		</a>
	</div>

	{#if stats}
		<!-- Stats Cards -->
		<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
			<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
				<div class="text-sm text-gray-500 mb-1">Gesamt</div>
				<div class="text-2xl font-bold text-gray-900">{stats.total}</div>
			</div>
			<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
				<div class="text-sm text-gray-500 mb-1">Bewilligt</div>
				<div class="text-2xl font-bold text-green-600">{stats.approved}</div>
			</div>
			<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
				<div class="text-sm text-gray-500 mb-1">Bewilligtes Volumen</div>
				<div class="text-2xl font-bold text-gray-900">{formatAmount(stats.total_approved)}</div>
			</div>
			<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
				<div class="text-sm text-gray-500 mb-1">Erfolgsquote</div>
				<div class="text-2xl font-bold text-blue-600">{stats.success_rate.toFixed(0)}%</div>
			</div>
		</div>

		<!-- Pipeline -->
		<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4 mb-8">
			<div class="flex items-center gap-2 mb-4">
				<TrendingUp class="w-5 h-5 text-gray-400" />
				<span class="font-medium text-gray-700">Pipeline</span>
			</div>
			<div class="flex items-center gap-2">
				<div class="flex-1 text-center p-3 rounded bg-gray-50">
					<div class="text-xs text-gray-500">Geplant</div>
					<div class="text-lg font-bold">{stats.planned}</div>
				</div>
				<ChevronRight class="w-4 h-4 text-gray-300" />
				<div class="flex-1 text-center p-3 rounded bg-blue-50">
					<div class="text-xs text-gray-500">Entwurf</div>
					<div class="text-lg font-bold">{stats.drafting}</div>
				</div>
				<ChevronRight class="w-4 h-4 text-gray-300" />
				<div class="flex-1 text-center p-3 rounded bg-amber-50">
					<div class="text-xs text-gray-500">Eingereicht</div>
					<div class="text-lg font-bold">{stats.submitted}</div>
				</div>
				<ChevronRight class="w-4 h-4 text-gray-300" />
				<div class="flex-1 text-center p-3 rounded bg-purple-50">
					<div class="text-xs text-gray-500">Prüfung</div>
					<div class="text-lg font-bold">{stats.in_review}</div>
				</div>
				<ChevronRight class="w-4 h-4 text-gray-300" />
				<div class="flex-1 text-center p-3 rounded bg-green-50">
					<div class="text-xs text-gray-500">Bewilligt</div>
					<div class="text-lg font-bold text-green-600">{stats.approved}</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- Filter -->
	<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4 mb-6">
		<div class="flex items-center gap-4">
			<Filter class="w-5 h-5 text-gray-400" />
			<select bind:value={statusFilter} on:change={loadData} class="select select-bordered select-sm">
				<option value="">Alle Status</option>
				{#each Object.entries(statusLabels) as [value, { label }]}
					<option value={value}>{label}</option>
				{/each}
			</select>
		</div>
	</div>

	<!-- List -->
	{#if loading}
		<div class="flex items-center justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if error}
		<div class="alert alert-error">{error}</div>
	{:else if antraege.length === 0}
		<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
			<FileText class="w-12 h-12 mx-auto mb-4 text-gray-400" />
			<h3 class="text-lg font-medium text-gray-900 mb-2">Noch keine Anträge</h3>
			<p class="text-gray-600 mb-4">Beginnen Sie mit der Förderungssuche um passende Programme zu finden.</p>
			<a href="/foerderungen/suche" class="btn btn-primary">Förderungssuche starten</a>
		</div>
	{:else}
		<div class="space-y-4">
			{#each antraege as antrag}
				{@const status = statusLabels[antrag.status] || statusLabels['planned']}
				<a href="/antraege/{antrag.id}" class="block bg-white rounded-lg shadow-sm border border-gray-200 hover:shadow-md transition-shadow">
					<div class="p-6">
						<div class="flex items-start justify-between">
							<div class="flex-1">
								<div class="flex items-center gap-2 mb-2">
									<span class="px-2 py-0.5 text-xs font-medium rounded-full {status.color} flex items-center gap-1">
										<svelte:component this={status.icon} class="w-3 h-3" />
										{status.label}
									</span>
									{#if antrag.internal_reference}
										<span class="text-xs text-gray-500">#{antrag.internal_reference}</span>
									{/if}
								</div>
								<h3 class="text-lg font-semibold text-gray-900">Antrag {antrag.id.slice(0, 8)}</h3>

								<div class="flex flex-wrap items-center gap-4 mt-3 text-sm text-gray-600">
									{#if antrag.requested_amount}
										<div class="flex items-center gap-1">
											<Euro class="w-4 h-4" />
											Beantragt: {formatAmount(antrag.requested_amount)}
										</div>
									{/if}
									{#if antrag.approved_amount}
										<div class="flex items-center gap-1 text-green-600">
											<CheckCircle class="w-4 h-4" />
											Bewilligt: {formatAmount(antrag.approved_amount)}
										</div>
									{/if}
									{#if antrag.submitted_at}
										<div class="flex items-center gap-1">
											<Calendar class="w-4 h-4" />
											Eingereicht: {new Date(antrag.submitted_at).toLocaleDateString('de-AT')}
										</div>
									{/if}
								</div>
							</div>
							<ChevronRight class="w-5 h-5 text-gray-400 flex-shrink-0 ml-4" />
						</div>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
