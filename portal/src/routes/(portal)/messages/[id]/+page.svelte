<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { ArrowLeft, Send } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { api } from '$lib/api/client';
	import { websocket } from '$lib/stores/websocket';
	import { auth } from '$lib/stores/auth';

	let thread: any = null;
	let messages: any[] = [];
	let loading = true;
	let error = '';
	let newMessage = '';
	let sending = false;
	let messagesContainer: HTMLDivElement;

	let unsubscribe: (() => void) | null = null;

	onMount(async () => {
		const id = $page.params.id;

		try {
			[thread, { messages: messages }] = await Promise.all([
				api.getThread(id),
				api.getMessages(id, { limit: 100 })
			]);

			// Mark as read
			await api.markAsRead(id);

			// Scroll to bottom
			setTimeout(scrollToBottom, 100);

			// Listen for new messages
			unsubscribe = websocket.on('new_message', (payload) => {
				if (payload.thread_id === thread.id) {
					messages = [...messages, payload.message];
					setTimeout(scrollToBottom, 100);
				}
			});
		} catch (e: any) {
			error = e.message || 'Konversation nicht gefunden';
		} finally {
			loading = false;
		}
	});

	onDestroy(() => {
		if (unsubscribe) unsubscribe();
	});

	async function sendMessage() {
		if (!newMessage.trim() || !thread) return;

		sending = true;
		try {
			const msg = await api.sendMessage(thread.id, newMessage);
			messages = [...messages, msg];
			newMessage = '';
			setTimeout(scrollToBottom, 100);
		} catch (e: any) {
			error = e.message || 'Senden fehlgeschlagen';
		} finally {
			sending = false;
		}
	}

	function scrollToBottom() {
		if (messagesContainer) {
			messagesContainer.scrollTop = messagesContainer.scrollHeight;
		}
	}

	function formatTime(date: string): string {
		return new Date(date).toLocaleTimeString('de-AT', {
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function formatDate(date: string): string {
		const d = new Date(date);
		const today = new Date();
		const yesterday = new Date(today);
		yesterday.setDate(yesterday.getDate() - 1);

		if (d.toDateString() === today.toDateString()) {
			return 'Heute';
		} else if (d.toDateString() === yesterday.toDateString()) {
			return 'Gestern';
		}

		return d.toLocaleDateString('de-AT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		});
	}

	function getMessageDate(msg: any, index: number): string | null {
		const currentDate = formatDate(msg.created_at);
		if (index === 0) return currentDate;

		const prevDate = formatDate(messages[index - 1].created_at);
		if (currentDate !== prevDate) return currentDate;

		return null;
	}

	function isOwnMessage(msg: any): boolean {
		return msg.sender_type === 'client';
	}
</script>

<svelte:head>
	<title>{thread?.subject || 'Nachricht'} | Mandantenportal</title>
</svelte:head>

<div class="flex flex-col h-[calc(100vh-8rem)]">
	<!-- Header -->
	<div class="flex items-center gap-4 mb-4">
		<button
			class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
			on:click={() => goto('/messages')}
		>
			<ArrowLeft class="w-5 h-5" />
		</button>
		<div>
			<h1 class="text-lg font-semibold text-gray-900">
				{thread?.subject || 'Laden...'}
			</h1>
		</div>
	</div>

	{#if loading}
		<Card class="flex-1">
			<div class="h-full flex items-center justify-center">
				<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
			</div>
		</Card>
	{:else if error && !thread}
		<Card class="flex-1 text-center py-12">
			<p class="text-red-600">{error}</p>
		</Card>
	{:else}
		<!-- Messages -->
		<Card padding="none" class="flex-1 flex flex-col overflow-hidden">
			<div
				bind:this={messagesContainer}
				class="flex-1 overflow-y-auto p-4 space-y-4"
			>
				{#each messages as msg, index}
					{@const dateLabel = getMessageDate(msg, index)}
					{@const own = isOwnMessage(msg)}

					{#if dateLabel}
						<div class="flex justify-center">
							<span class="text-xs text-gray-500 bg-gray-100 px-3 py-1 rounded-full">
								{dateLabel}
							</span>
						</div>
					{/if}

					<div class="flex" class:justify-end={own}>
						<div
							class="max-w-[75%] rounded-lg px-4 py-2"
							class:bg-primary={own}
							class:text-white={own}
							class:bg-gray-100={!own}
							class:text-gray-900={!own}
						>
							{#if !own && msg.sender_name}
								<p class="text-xs font-medium mb-1 opacity-75">
									{msg.sender_name}
								</p>
							{/if}
							<p class="whitespace-pre-wrap">{msg.content}</p>
							<p class="text-xs mt-1 opacity-75">
								{formatTime(msg.created_at)}
							</p>
						</div>
					</div>
				{/each}

				{#if messages.length === 0}
					<div class="text-center text-gray-500 py-8">
						Noch keine Nachrichten
					</div>
				{/if}
			</div>

			<!-- Input -->
			<div class="border-t border-gray-200 p-4">
				{#if error}
					<div class="bg-red-50 border border-red-200 text-red-700 px-3 py-2 rounded-lg text-sm mb-3">
						{error}
					</div>
				{/if}

				<form on:submit|preventDefault={sendMessage} class="flex gap-3">
					<input
						type="text"
						bind:value={newMessage}
						placeholder="Nachricht schreiben..."
						class="flex-1 rounded-lg border border-gray-300 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
					/>
					<Button
						type="submit"
						loading={sending}
						disabled={!newMessage.trim()}
					>
						<Send class="w-4 h-4" />
					</Button>
				</form>
			</div>
		</Card>
	{/if}
</div>
