<script lang="ts">
	import { onMount } from 'svelte';
	import { MessageSquare, Plus } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { api } from '$lib/api/client';

	let threads: any[] = [];
	let loading = true;
	let showNewThread = false;
	let newSubject = '';
	let newMessage = '';
	let creating = false;

	onMount(async () => {
		await loadThreads();
	});

	async function loadThreads() {
		loading = true;
		try {
			const result = await api.getThreads({ limit: 50 });
			threads = result.threads || [];
		} catch (e) {
			console.error('Failed to load threads:', e);
		} finally {
			loading = false;
		}
	}

	async function createThread() {
		if (!newSubject.trim() || !newMessage.trim()) return;

		creating = true;
		try {
			await api.startThread(newSubject, newMessage);
			showNewThread = false;
			newSubject = '';
			newMessage = '';
			await loadThreads();
		} catch (e) {
			console.error('Failed to create thread:', e);
		} finally {
			creating = false;
		}
	}

	function formatDate(date: string): string {
		const d = new Date(date);
		const now = new Date();
		const diff = now.getTime() - d.getTime();

		if (diff < 60000) return 'Gerade eben';
		if (diff < 3600000) return `vor ${Math.floor(diff / 60000)} Min.`;
		if (diff < 86400000) return `vor ${Math.floor(diff / 3600000)} Std.`;

		return d.toLocaleDateString('de-AT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		});
	}
</script>

<svelte:head>
	<title>Nachrichten | Mandantenportal</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-gray-900">Nachrichten</h1>

		<Button on:click={() => (showNewThread = true)}>
			<Plus class="w-4 h-4 mr-2" />
			Neue Nachricht
		</Button>
	</div>

	{#if loading}
		<div class="space-y-4">
			{#each [1, 2, 3] as _}
				<Card>
					<div class="h-20 bg-gray-100 rounded animate-pulse"></div>
				</Card>
			{/each}
		</div>
	{:else if threads.length === 0}
		<Card class="text-center py-12">
			<MessageSquare class="w-16 h-16 mx-auto mb-4 text-gray-300" />
			<h2 class="text-xl font-semibold text-gray-900 mb-2">Keine Nachrichten</h2>
			<p class="text-gray-600 mb-6">
				Starten Sie eine Unterhaltung mit Ihrem Steuerberater.
			</p>
			<Button on:click={() => (showNewThread = true)}>
				<Plus class="w-4 h-4 mr-2" />
				Neue Nachricht
			</Button>
		</Card>
	{:else}
		<div class="space-y-3">
			{#each threads as thread}
				<Card padding="none">
					<a
						href="/messages/{thread.id}"
						class="flex items-center gap-4 p-4 hover:bg-gray-50 transition-colors"
					>
						<div class="p-3 rounded-lg" class:bg-primary/10={thread.unread_count > 0} class:bg-gray-100={thread.unread_count === 0}>
							<MessageSquare class="w-6 h-6" class:text-primary={thread.unread_count > 0} class:text-gray-600={thread.unread_count === 0} />
						</div>

						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2">
								<p class="font-medium text-gray-900 truncate" class:font-semibold={thread.unread_count > 0}>
									{thread.subject}
								</p>
								{#if thread.unread_count > 0}
									<Badge variant="info">{thread.unread_count} neu</Badge>
								{/if}
							</div>
							{#if thread.last_message}
								<p class="text-sm text-gray-500 truncate">
									{thread.last_message}
								</p>
							{/if}
						</div>

						<div class="text-sm text-gray-500 whitespace-nowrap">
							{formatDate(thread.last_message_at || thread.created_at)}
						</div>
					</a>
				</Card>
			{/each}
		</div>
	{/if}
</div>

<!-- New thread modal -->
{#if showNewThread}
	<div
		class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4"
		on:click={() => (showNewThread = false)}
		on:keydown={(e) => e.key === 'Escape' && (showNewThread = false)}
		role="dialog"
	>
		<div
			class="bg-white rounded-lg shadow-xl max-w-md w-full p-6"
			on:click|stopPropagation
			role="document"
		>
			<h2 class="text-xl font-bold text-gray-900 mb-4">Neue Nachricht</h2>

			<div class="space-y-4">
				<div>
					<label class="block text-sm font-medium text-gray-700 mb-1">
						Betreff
					</label>
					<input
						type="text"
						bind:value={newSubject}
						placeholder="Worum geht es?"
						class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
					/>
				</div>

				<div>
					<label class="block text-sm font-medium text-gray-700 mb-1">
						Nachricht
					</label>
					<textarea
						bind:value={newMessage}
						rows="4"
						placeholder="Ihre Nachricht..."
						class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary resize-none"
					></textarea>
				</div>
			</div>

			<div class="flex gap-3 mt-6">
				<Button
					variant="outline"
					class="flex-1"
					on:click={() => (showNewThread = false)}
				>
					Abbrechen
				</Button>
				<Button
					class="flex-1"
					loading={creating}
					disabled={!newSubject.trim() || !newMessage.trim()}
					on:click={createThread}
				>
					Senden
				</Button>
			</div>
		</div>
	</div>
{/if}
