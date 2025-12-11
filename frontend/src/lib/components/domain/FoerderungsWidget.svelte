<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { formatCurrency } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import type { AntragStats, Notification as BaseNotification, NotificationListResponse } from '$lib/api/foerderung.types';

	// Display interface for widget matches
	interface WidgetMatch {
		foerderung_id: string;
		name: string;
		provider: string;
		total_score: number;
		max_amount?: number;
		deadline?: string;
	}

	interface MatchesResponse {
		matches: WidgetMatch[];
	}

	let loading = $state(true);
	let matches = $state<WidgetMatch[]>([]);
	let stats = $state<AntragStats | null>(null);
	let notifications = $state<BaseNotification[]>([]);

	async function loadData() {
		loading = true;
		try {
			const [matchesRes, statsRes, notifRes] = await Promise.all([
				api.get<MatchesResponse>('/api/v1/foerderungen/matches?limit=3').catch(() => ({ matches: [] })),
				api.get<AntragStats>('/api/v1/antraege/stats').catch(() => null),
				api.get<NotificationListResponse>('/api/v1/monitor/notifications?limit=3').catch(() => ({ notifications: [] }))
			]);

			matches = matchesRes?.matches || [];
			stats = statsRes;
			notifications = notifRes?.notifications || [];
		} catch (e) {
			console.error('Failed to load Förderungen widget data:', e);
		} finally {
			loading = false;
		}
	}

	function formatAmount(amount?: number): string {
		if (!amount) return '-';
		return formatCurrency(amount, 'EUR', { maximumFractionDigits: 0 });
	}

	function formatDeadline(deadline?: string): string {
		if (!deadline) return 'Laufend';
		const date = new Date(deadline);
		const daysLeft = Math.ceil((date.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
		if (daysLeft < 0) return 'Abgelaufen';
		if (daysLeft <= 7) return `${daysLeft} Tage`;
		return date.toLocaleDateString('de-AT');
	}

	function getScoreVariant(score: number): 'success' | 'warning' | 'default' {
		if (score >= 80) return 'success';
		if (score >= 60) return 'warning';
		return 'default';
	}

	onMount(loadData);
</script>

<!-- Stats Row -->
{#if stats}
	<div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
		<Card class="stagger-1">
			<div class="flex items-center justify-between">
				<div>
					<p class="text-sm text-[var(--color-ink-muted)]">Anträge</p>
					<p class="text-3xl font-bold text-[var(--color-ink)] mt-1">{stats.total}</p>
				</div>
				<div class="w-12 h-12 rounded-xl bg-[var(--color-info-muted)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-info)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect width="8" height="4" x="8" y="2" rx="1" ry="1"/>
						<path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/>
						<path d="M12 11h4"/><path d="M12 16h4"/><path d="M8 11h.01"/><path d="M8 16h.01"/>
					</svg>
				</div>
			</div>
		</Card>

		<Card class="stagger-2">
			<div class="flex items-center justify-between">
				<div>
					<p class="text-sm text-[var(--color-ink-muted)]">In Bearbeitung</p>
					<p class="text-3xl font-bold text-[var(--color-warning)] mt-1">{stats.in_progress ?? (stats.drafting + stats.submitted + stats.in_review)}</p>
				</div>
				<div class="w-12 h-12 rounded-xl bg-[var(--color-warning-muted)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-warning)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10"/>
						<polyline points="12 6 12 12 16 14"/>
					</svg>
				</div>
			</div>
		</Card>

		<Card class="stagger-3">
			<div class="flex items-center justify-between">
				<div>
					<p class="text-sm text-[var(--color-ink-muted)]">Bewilligt</p>
					<p class="text-3xl font-bold text-[var(--color-success)] mt-1">{stats.approved}</p>
				</div>
				<div class="w-12 h-12 rounded-xl bg-[var(--color-success-muted)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<polyline points="20 6 9 17 4 12"/>
					</svg>
				</div>
			</div>
		</Card>

		<Card class="stagger-4">
			<div class="flex items-center justify-between">
				<div>
					<p class="text-sm text-[var(--color-ink-muted)]">Bewilligt</p>
					<p class="text-2xl font-bold text-[var(--color-ink)] mt-1">{formatAmount(stats.total_approved)}</p>
				</div>
				<div class="w-12 h-12 rounded-xl bg-[var(--color-accent-muted)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M4 10h12"/><path d="M4 14h9"/>
						<path d="M19 6a7.7 7.7 0 0 0-5.2-2A7.9 7.9 0 0 0 6 12c0 4.4 3.5 8 7.8 8 2 0 3.8-.8 5.2-2"/>
					</svg>
				</div>
			</div>
		</Card>
	</div>
{/if}

<!-- Main Widgets Grid -->
<div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
	<!-- Top Matches -->
	<Card padding="none">
		<div class="p-4 border-b border-black/6 flex items-center justify-between">
			<h2 class="font-semibold text-[var(--color-ink)]">Passende Förderungen</h2>
			<Button variant="ghost" size="sm" href="/foerderungen/suche">
				Suche starten
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M5 12h14M12 5l7 7-7 7"/>
				</svg>
			</Button>
		</div>
		{#if loading}
			<div class="p-8 flex justify-center">
				<span class="loading loading-spinner loading-md"></span>
			</div>
		{:else if matches.length === 0}
			<div class="p-8 text-center">
				<div class="w-12 h-12 mx-auto mb-3 rounded-xl bg-[var(--color-paper-inset)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/>
					</svg>
				</div>
				<p class="text-sm text-[var(--color-ink-muted)]">Erstellen Sie ein Profil um passende Förderungen zu finden.</p>
				<Button variant="secondary" size="sm" href="/profile/new" class="mt-3">
					Profil erstellen
				</Button>
			</div>
		{:else}
			<div class="divide-y divide-black/4">
				{#each matches as match}
					<a href="/foerderungen/{match.foerderung_id}" class="flex items-center gap-4 p-4 hover:bg-[var(--color-paper-inset)] transition-colors">
						<div class="w-10 h-10 rounded-lg bg-[var(--color-accent-muted)] flex items-center justify-center flex-shrink-0">
							<svg class="w-5 h-5 text-[var(--color-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M4 10h12"/><path d="M4 14h9"/>
								<path d="M19 6a7.7 7.7 0 0 0-5.2-2A7.9 7.9 0 0 0 6 12c0 4.4 3.5 8 7.8 8 2 0 3.8-.8 5.2-2"/>
							</svg>
						</div>
						<div class="flex-1 min-w-0">
							<p class="text-sm font-medium text-[var(--color-ink)] truncate">{match.name}</p>
							<p class="text-xs text-[var(--color-ink-muted)] mt-0.5">{match.provider}</p>
						</div>
						<div class="text-right flex-shrink-0">
							<Badge variant={getScoreVariant(match.total_score)} size="sm">
								{match.total_score}%
							</Badge>
							<p class="text-xs text-[var(--color-ink-muted)] mt-1">
								{formatDeadline(match.deadline)}
							</p>
						</div>
					</a>
				{/each}
			</div>
		{/if}
	</Card>

	<!-- Notifications -->
	<Card padding="none">
		<div class="p-4 border-b border-black/6 flex items-center justify-between">
			<h2 class="font-semibold text-[var(--color-ink)]">Benachrichtigungen</h2>
			<Button variant="ghost" size="sm" href="/foerderungen">
				Alle anzeigen
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M5 12h14M12 5l7 7-7 7"/>
				</svg>
			</Button>
		</div>
		{#if loading}
			<div class="p-8 flex justify-center">
				<span class="loading loading-spinner loading-md"></span>
			</div>
		{:else if notifications.length === 0}
			<div class="p-8 text-center">
				<div class="w-12 h-12 mx-auto mb-3 rounded-xl bg-[var(--color-paper-inset)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/>
						<path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>
					</svg>
				</div>
				<p class="text-sm text-[var(--color-ink-muted)]">Keine neuen Benachrichtigungen</p>
			</div>
		{:else}
			<div class="divide-y divide-black/4">
				{#each notifications as notif}
					<div class="p-4">
						<div class="flex items-start gap-3">
							<span class="w-2 h-2 rounded-full bg-[var(--color-accent)] mt-2 flex-shrink-0"></span>
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium text-[var(--color-ink)]">{notif.foerderung_name}</p>
								<p class="text-xs text-[var(--color-ink-muted)] mt-0.5 truncate">{notif.message}</p>
								<p class="text-xs text-[var(--color-ink-muted)] mt-1">
									{new Date(notif.created_at).toLocaleDateString('de-AT')}
								</p>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</Card>
</div>

<!-- Quick Actions -->
<Card class="mt-6 bg-[var(--color-accent-muted)]">
	<div class="flex items-center justify-between flex-wrap gap-4">
		<div>
			<h3 class="font-semibold text-[var(--color-ink)]">Förderungen entdecken</h3>
			<p class="text-sm text-[var(--color-ink-muted)] mt-1">
				Finden Sie passende Förderungen für Ihr Unternehmen mit unserer KI-gestützten Suche.
			</p>
		</div>
		<div class="flex gap-3">
			<Button variant="secondary" href="/profile">
				Profile verwalten
			</Button>
			<Button href="/foerderungen/suche">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/>
				</svg>
				Förderungssuche
			</Button>
		</div>
	</div>
</Card>
