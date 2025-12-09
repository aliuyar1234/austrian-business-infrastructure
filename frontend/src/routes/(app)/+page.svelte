<script lang="ts">
	import { onMount } from 'svelte';
	import { user } from '$lib/stores/auth';
	import { formatDate, formatDateTime } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	// Mock data - would come from API
	let stats = $state({
		newDocuments: 12,
		pendingDeadlines: 3,
		activeAccounts: 8,
		syncStatus: 'healthy' as 'healthy' | 'syncing' | 'error'
	});

	let recentDocuments = $state([
		{ id: '1', title: 'Bescheid Umsatzsteuer 2024', account: 'Muster GmbH', date: new Date(), type: 'bescheid', unread: true },
		{ id: '2', title: 'Ergänzungsersuchen', account: 'Test AG', date: new Date(Date.now() - 86400000), type: 'ersuchen', unread: true },
		{ id: '3', title: 'Vorauszahlungsbescheid', account: 'Muster GmbH', date: new Date(Date.now() - 172800000), type: 'bescheid', unread: false },
	]);

	let upcomingDeadlines = $state([
		{ id: '1', title: 'UVA Jänner 2025', account: 'Muster GmbH', dueDate: new Date(Date.now() + 5 * 86400000), daysLeft: 5 },
		{ id: '2', title: 'Ergänzungsersuchen Frist', account: 'Test AG', dueDate: new Date(Date.now() + 10 * 86400000), daysLeft: 10 },
		{ id: '3', title: 'ZM Q4 2024', account: 'Muster GmbH', dueDate: new Date(Date.now() + 15 * 86400000), daysLeft: 15 },
	]);

	let accountActivity = $state([
		{ id: '1', account: 'Muster GmbH', action: 'Sync completed', time: '2 min ago', status: 'success' as const },
		{ id: '2', account: 'Test AG', action: 'New document received', time: '15 min ago', status: 'info' as const },
		{ id: '3', account: 'Demo GmbH', action: 'Connection error', time: '1 hour ago', status: 'error' as const },
	]);

	function getDeadlineUrgency(daysLeft: number): 'error' | 'warning' | 'info' {
		if (daysLeft <= 3) return 'error';
		if (daysLeft <= 7) return 'warning';
		return 'info';
	}
</script>

<div class="max-w-7xl mx-auto space-y-8 animate-in">
	<!-- Welcome section -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-[var(--color-ink)]">
				Guten Tag{$user ? `, ${$user.name.split(' ')[0]}` : ''}
			</h1>
			<p class="text-[var(--color-ink-muted)] mt-1">
				Here's what's happening with your accounts today.
			</p>
		</div>
		<div class="flex items-center gap-3">
			<Button variant="secondary" href="/accounts">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 5v14M5 12h14"/>
				</svg>
				Add Account
			</Button>
			<Button href="/uva/new">
				Create UVA
			</Button>
		</div>
	</div>

	<!-- Stats grid -->
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
		<Card class="stagger-1">
			<div class="flex items-center justify-between">
				<div>
					<p class="text-sm text-[var(--color-ink-muted)]">New Documents</p>
					<p class="text-3xl font-bold text-[var(--color-ink)] mt-1">{stats.newDocuments}</p>
				</div>
				<div class="w-12 h-12 rounded-xl bg-[var(--color-info-muted)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-info)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
						<polyline points="14 2 14 8 20 8"/>
					</svg>
				</div>
			</div>
		</Card>

		<Card class="stagger-2">
			<div class="flex items-center justify-between">
				<div>
					<p class="text-sm text-[var(--color-ink-muted)]">Pending Deadlines</p>
					<p class="text-3xl font-bold text-[var(--color-ink)] mt-1">{stats.pendingDeadlines}</p>
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
					<p class="text-sm text-[var(--color-ink-muted)]">Active Accounts</p>
					<p class="text-3xl font-bold text-[var(--color-ink)] mt-1">{stats.activeAccounts}</p>
				</div>
				<div class="w-12 h-12 rounded-xl bg-[var(--color-success-muted)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect width="16" height="20" x="4" y="2" rx="2" ry="2"/>
						<path d="M9 22v-4h6v4"/>
						<path d="M8 6h.01"/><path d="M16 6h.01"/><path d="M12 6h.01"/>
						<path d="M12 10h.01"/><path d="M12 14h.01"/>
						<path d="M16 10h.01"/><path d="M16 14h.01"/>
						<path d="M8 10h.01"/><path d="M8 14h.01"/>
					</svg>
				</div>
			</div>
		</Card>

		<Card class="stagger-4">
			<div class="flex items-center justify-between">
				<div>
					<p class="text-sm text-[var(--color-ink-muted)]">Sync Status</p>
					<div class="flex items-center gap-2 mt-2">
						<span class="status-dot status-dot-success"></span>
						<span class="text-sm font-medium text-[var(--color-ink)]">All synced</span>
					</div>
				</div>
				<div class="w-12 h-12 rounded-xl bg-[var(--color-success-muted)] flex items-center justify-center">
					<svg class="w-6 h-6 text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<polyline points="20 6 9 17 4 12"/>
					</svg>
				</div>
			</div>
		</Card>
	</div>

	<!-- Main content grid -->
	<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
		<!-- Recent documents -->
		<Card padding="none" class="lg:col-span-2">
			<div class="p-4 border-b border-black/6 flex items-center justify-between">
				<h2 class="font-semibold text-[var(--color-ink)]">Recent Documents</h2>
				<Button variant="ghost" size="sm" href="/documents">
					View all
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M5 12h14M12 5l7 7-7 7"/>
					</svg>
				</Button>
			</div>
			<div class="divide-y divide-black/4">
				{#each recentDocuments as doc}
					<a href="/documents/{doc.id}" class="flex items-center gap-4 p-4 hover:bg-[var(--color-paper-inset)] transition-colors">
						<div class="w-10 h-10 rounded-lg bg-[var(--color-paper-inset)] flex items-center justify-center flex-shrink-0">
							<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
								<polyline points="14 2 14 8 20 8"/>
							</svg>
						</div>
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2">
								<p class="text-sm font-medium text-[var(--color-ink)] truncate">{doc.title}</p>
								{#if doc.unread}
									<span class="w-2 h-2 rounded-full bg-[var(--color-accent)]"></span>
								{/if}
							</div>
							<p class="text-xs text-[var(--color-ink-muted)] mt-0.5">{doc.account}</p>
						</div>
						<div class="text-right">
							<p class="text-xs text-[var(--color-ink-muted)]">{formatDate(doc.date)}</p>
							<Badge size="sm" variant={doc.type === 'ersuchen' ? 'warning' : 'default'} class="mt-1">
								{doc.type}
							</Badge>
						</div>
					</a>
				{/each}
			</div>
		</Card>

		<!-- Upcoming deadlines -->
		<Card padding="none">
			<div class="p-4 border-b border-black/6 flex items-center justify-between">
				<h2 class="font-semibold text-[var(--color-ink)]">Upcoming Deadlines</h2>
				<Button variant="ghost" size="sm" href="/calendar">
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M8 2v4M16 2v4"/>
						<rect width="18" height="18" x="3" y="4" rx="2"/>
						<path d="M3 10h18"/>
					</svg>
				</Button>
			</div>
			<div class="divide-y divide-black/4">
				{#each upcomingDeadlines as deadline}
					{@const urgency = getDeadlineUrgency(deadline.daysLeft)}
					<div class="p-4">
						<div class="flex items-start justify-between">
							<div>
								<p class="text-sm font-medium text-[var(--color-ink)]">{deadline.title}</p>
								<p class="text-xs text-[var(--color-ink-muted)] mt-0.5">{deadline.account}</p>
							</div>
							<Badge variant={urgency} dot>
								{deadline.daysLeft} {deadline.daysLeft === 1 ? 'day' : 'days'}
							</Badge>
						</div>
					</div>
				{/each}
			</div>
		</Card>
	</div>

	<!-- Activity feed -->
	<Card padding="none">
		<div class="p-4 border-b border-black/6">
			<h2 class="font-semibold text-[var(--color-ink)]">Recent Activity</h2>
		</div>
		<div class="divide-y divide-black/4">
			{#each accountActivity as activity}
				<div class="flex items-center gap-4 p-4">
					<div class={`w-2 h-2 rounded-full ${
						activity.status === 'success' ? 'bg-[var(--color-success)]' :
						activity.status === 'error' ? 'bg-[var(--color-error)]' :
						'bg-[var(--color-info)]'
					}`}></div>
					<div class="flex-1">
						<p class="text-sm text-[var(--color-ink)]">
							<span class="font-medium">{activity.account}</span>
							<span class="text-[var(--color-ink-muted)]"> — {activity.action}</span>
						</p>
					</div>
					<p class="text-xs text-[var(--color-ink-muted)]">{activity.time}</p>
				</div>
			{/each}
		</div>
	</Card>
</div>
