<script lang="ts">
	import { onMount } from 'svelte';
	import { FileText, CheckCircle, ListTodo, Upload, MessageSquare, AlertCircle } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { api } from '$lib/api/client';
	import { branding } from '$lib/stores/branding';

	let loading = true;
	let stats = {
		pendingApprovals: 0,
		openTasks: 0,
		unreadMessages: 0,
		recentDocuments: [] as any[]
	};

	onMount(async () => {
		try {
			const [approvals, tasks, messages, documents] = await Promise.all([
				api.getApprovals({ status: 'pending', limit: 5 }),
				api.getTasks({ status: 'open', limit: 5 }),
				api.getUnreadCount(),
				api.getDocuments({ limit: 5 })
			]);

			stats = {
				pendingApprovals: approvals.total,
				openTasks: tasks.total,
				unreadMessages: messages.unread_count,
				recentDocuments: documents.documents || []
			};
		} catch (e) {
			console.error('Failed to load dashboard:', e);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Dashboard | {$branding.company_name}</title>
</svelte:head>

<div class="space-y-6">
	<!-- Welcome message -->
	{#if $branding.welcome_message}
		<Card>
			<p class="text-gray-700">{$branding.welcome_message}</p>
		</Card>
	{/if}

	<!-- Stats -->
	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
		<Card>
			<div class="flex items-center gap-4">
				<div class="p-3 bg-yellow-100 rounded-lg">
					<CheckCircle class="w-6 h-6 text-yellow-600" />
				</div>
				<div>
					<p class="text-2xl font-bold text-gray-900">
						{loading ? '-' : stats.pendingApprovals}
					</p>
					<p class="text-sm text-gray-500">Offene Freigaben</p>
				</div>
			</div>
		</Card>

		<Card>
			<div class="flex items-center gap-4">
				<div class="p-3 bg-blue-100 rounded-lg">
					<ListTodo class="w-6 h-6 text-blue-600" />
				</div>
				<div>
					<p class="text-2xl font-bold text-gray-900">
						{loading ? '-' : stats.openTasks}
					</p>
					<p class="text-sm text-gray-500">Offene Aufgaben</p>
				</div>
			</div>
		</Card>

		<Card>
			<div class="flex items-center gap-4">
				<div class="p-3 bg-green-100 rounded-lg">
					<MessageSquare class="w-6 h-6 text-green-600" />
				</div>
				<div>
					<p class="text-2xl font-bold text-gray-900">
						{loading ? '-' : stats.unreadMessages}
					</p>
					<p class="text-sm text-gray-500">Ungelesene Nachrichten</p>
				</div>
			</div>
		</Card>

		<Card>
			<a href="/upload" class="flex items-center gap-4 group">
				<div class="p-3 bg-primary/10 rounded-lg group-hover:bg-primary/20 transition-colors">
					<Upload class="w-6 h-6 text-primary" />
				</div>
				<div>
					<p class="font-medium text-gray-900 group-hover:text-primary">Beleg hochladen</p>
					<p class="text-sm text-gray-500">Dokument einreichen</p>
				</div>
			</a>
		</Card>
	</div>

	<!-- Quick actions -->
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
		<!-- Pending approvals -->
		<Card>
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-semibold text-gray-900">Offene Freigaben</h2>
				<a href="/approvals" class="text-sm text-primary hover:underline">Alle anzeigen</a>
			</div>

			{#if loading}
				<div class="space-y-3">
					{#each [1, 2, 3] as _}
						<div class="h-12 bg-gray-100 rounded animate-pulse"></div>
					{/each}
				</div>
			{:else if stats.pendingApprovals === 0}
				<div class="text-center py-8 text-gray-500">
					<CheckCircle class="w-12 h-12 mx-auto mb-2 text-green-500" />
					<p>Keine offenen Freigaben</p>
				</div>
			{:else}
				<div class="space-y-2">
					<a href="/approvals" class="block p-3 bg-yellow-50 border border-yellow-200 rounded-lg hover:bg-yellow-100 transition-colors">
						<div class="flex items-center gap-3">
							<AlertCircle class="w-5 h-5 text-yellow-600" />
							<span class="text-sm text-gray-900">
								{stats.pendingApprovals} Dokument{stats.pendingApprovals !== 1 ? 'e' : ''} warten auf Ihre Freigabe
							</span>
						</div>
					</a>
				</div>
			{/if}
		</Card>

		<!-- Recent documents -->
		<Card>
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-semibold text-gray-900">Aktuelle Dokumente</h2>
				<a href="/documents" class="text-sm text-primary hover:underline">Alle anzeigen</a>
			</div>

			{#if loading}
				<div class="space-y-3">
					{#each [1, 2, 3] as _}
						<div class="h-12 bg-gray-100 rounded animate-pulse"></div>
					{/each}
				</div>
			{:else if stats.recentDocuments.length === 0}
				<div class="text-center py-8 text-gray-500">
					<FileText class="w-12 h-12 mx-auto mb-2" />
					<p>Noch keine Dokumente</p>
				</div>
			{:else}
				<div class="space-y-2">
					{#each stats.recentDocuments as doc}
						<a
							href="/documents/{doc.id}"
							class="flex items-center gap-3 p-3 hover:bg-gray-50 rounded-lg transition-colors"
						>
							<FileText class="w-5 h-5 text-gray-400" />
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium text-gray-900 truncate">{doc.title || doc.file_name}</p>
								<p class="text-xs text-gray-500">
									{new Date(doc.shared_at || doc.created_at).toLocaleDateString('de-AT')}
								</p>
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</Card>
	</div>
</div>
