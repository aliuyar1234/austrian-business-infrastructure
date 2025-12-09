<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { accounts, accountsList, getAccountTypeLabel, getAccountStatusLabel, type Account, type UpdateAccountRequest } from '$lib/stores/accounts';
	import { formatDate, formatDateTime } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	let account = $derived($accountsList.find(a => a.id === $page.params.id));
	let isEditing = $state(false);
	let isSyncing = $state(false);
	let isTesting = $state(false);
	let isDeleting = $state(false);
	let showDeleteConfirm = $state(false);

	// Edit form state
	let editName = $state('');
	let editPin = $state('');
	let editPassword = $state('');
	let editTags = $state('');

	onMount(() => {
		accounts.load();
	});

	$effect(() => {
		if (account) {
			editName = account.name;
			editTags = account.tags.join(', ');
		}
	});

	function startEditing() {
		if (account) {
			editName = account.name;
			editTags = account.tags.join(', ');
			editPin = '';
			editPassword = '';
			isEditing = true;
		}
	}

	function cancelEditing() {
		isEditing = false;
		editPin = '';
		editPassword = '';
	}

	async function saveChanges() {
		if (!account) return;

		const request: UpdateAccountRequest = {
			name: editName !== account.name ? editName : undefined,
			tags: editTags.split(',').map(t => t.trim()).filter(Boolean),
		};

		if (account.type === 'finanzonline' && editPin) {
			request.pin = editPin;
		} else if (account.type === 'elda' && editPassword) {
			request.eldaPassword = editPassword;
		} else if (account.type === 'firmenbuch' && editPassword) {
			request.password = editPassword;
		}

		const result = await accounts.update(account.id, request);
		if (result.success) {
			isEditing = false;
		}
	}

	async function handleSync() {
		if (!account) return;
		isSyncing = true;
		await accounts.sync(account.id);
		isSyncing = false;
	}

	async function handleTest() {
		if (!account) return;
		isTesting = true;
		await accounts.testConnection(account.id);
		isTesting = false;
	}

	async function handleDelete() {
		if (!account) return;
		isDeleting = true;
		const result = await accounts.remove(account.id);
		if (result.success) {
			goto('/accounts');
		}
		isDeleting = false;
	}

	function getStatusVariant(status: Account['status']): 'success' | 'warning' | 'error' | 'info' {
		switch (status) {
			case 'active': return 'success';
			case 'syncing': return 'info';
			case 'error': return 'error';
			case 'pending': return 'warning';
		}
	}
</script>

<svelte:head>
	<title>{account?.name ?? 'Account'} - Austrian Business Infrastructure</title>
</svelte:head>

{#if !account}
	<div class="max-w-4xl mx-auto animate-in">
		<Card class="text-center py-12">
			<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<circle cx="12" cy="12" r="10"/>
				<path d="M12 8v4M12 16h.01"/>
			</svg>
			<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">Account not found</h3>
			<p class="mt-2 text-[var(--color-ink-muted)]">The account you're looking for doesn't exist.</p>
			<Button variant="secondary" href="/accounts" class="mt-6">Back to accounts</Button>
		</Card>
	</div>
{:else}
	<div class="max-w-4xl mx-auto space-y-6 animate-in">
		<!-- Header -->
		<div class="flex items-start justify-between">
			<div class="flex items-center gap-4">
				<a href="/accounts" aria-label="Back to accounts" class="w-10 h-10 flex items-center justify-center rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors">
					<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="m12 19-7-7 7-7M19 12H5"/>
					</svg>
				</a>
				<div>
					<div class="flex items-center gap-3">
						<h1 class="text-xl font-semibold text-[var(--color-ink)]">{account.name}</h1>
						<Badge variant={getStatusVariant(account.status)} dot>
							{getAccountStatusLabel(account.status)}
						</Badge>
					</div>
					<p class="text-sm text-[var(--color-ink-muted)] mt-0.5">
						{getAccountTypeLabel(account.type)}
						{#if account.teilnehmerId}
							• TID: {account.teilnehmerId}
						{/if}
					</p>
				</div>
			</div>
			<div class="flex items-center gap-2">
				<Button variant="secondary" onclick={handleTest} loading={isTesting}>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/>
						<path d="m9 12 2 2 4-4"/>
					</svg>
					Test
				</Button>
				<Button onclick={handleSync} loading={isSyncing}>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
						<path d="M3 3v5h5"/>
						<path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/>
						<path d="M16 16h5v5"/>
					</svg>
					Sync
				</Button>
			</div>
		</div>

		<!-- Account details card -->
		<Card padding="none">
			<div class="p-4 border-b border-black/6 flex items-center justify-between">
				<h2 class="font-semibold text-[var(--color-ink)]">Account Details</h2>
				{#if !isEditing}
					<Button variant="ghost" size="sm" onclick={startEditing}>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/>
							<path d="m15 5 4 4"/>
						</svg>
						Edit
					</Button>
				{/if}
			</div>

			{#if isEditing}
				<form onsubmit={(e) => { e.preventDefault(); saveChanges(); }} class="p-4 space-y-4">
					<div>
						<label for="edit-name" class="label">Account name</label>
						<Input type="text" id="edit-name" bind:value={editName} />
					</div>

					{#if account.type === 'finanzonline'}
						<div class="grid sm:grid-cols-2 gap-4">
							<div>
								<label for="readonly-tid" class="label">Teilnehmer-ID</label>
								<Input type="text" id="readonly-tid" value={account.teilnehmerId} disabled />
								<p class="mt-1 text-xs text-[var(--color-ink-muted)]">Cannot be changed</p>
							</div>
							<div>
								<label for="readonly-bid" class="label">Benutzer-ID</label>
								<Input type="text" id="readonly-bid" value={account.benutzerId} disabled />
								<p class="mt-1 text-xs text-[var(--color-ink-muted)]">Cannot be changed</p>
							</div>
						</div>
						<div>
							<label for="edit-pin" class="label">New PIN (leave empty to keep current)</label>
							<Input type="password" id="edit-pin" bind:value={editPin} placeholder="Enter new PIN" />
						</div>
					{:else}
						<div>
							<label for="edit-password" class="label">New password (leave empty to keep current)</label>
							<Input type="password" id="edit-password" bind:value={editPassword} placeholder="Enter new password" />
						</div>
					{/if}

					<div>
						<label for="edit-tags" class="label">Tags</label>
						<Input type="text" id="edit-tags" bind:value={editTags} placeholder="Comma-separated tags" />
					</div>

					<div class="flex justify-end gap-2 pt-2">
						<Button variant="secondary" type="button" onclick={cancelEditing}>Cancel</Button>
						<Button type="submit">Save changes</Button>
					</div>
				</form>
			{:else}
				<div class="divide-y divide-black/4">
					<div class="flex items-center justify-between px-4 py-3">
						<span class="text-sm text-[var(--color-ink-muted)]">Type</span>
						<span class="text-sm font-medium text-[var(--color-ink)]">{getAccountTypeLabel(account.type)}</span>
					</div>
					{#if account.teilnehmerId}
						<div class="flex items-center justify-between px-4 py-3">
							<span class="text-sm text-[var(--color-ink-muted)]">Teilnehmer-ID</span>
							<span class="text-sm font-mono text-[var(--color-ink)]">{account.teilnehmerId}</span>
						</div>
					{/if}
					{#if account.benutzerId}
						<div class="flex items-center justify-between px-4 py-3">
							<span class="text-sm text-[var(--color-ink-muted)]">Benutzer-ID</span>
							<span class="text-sm font-mono text-[var(--color-ink)]">{account.benutzerId}</span>
						</div>
					{/if}
					{#if account.dienstgeberId}
						<div class="flex items-center justify-between px-4 py-3">
							<span class="text-sm text-[var(--color-ink-muted)]">Dienstgeber-ID</span>
							<span class="text-sm font-mono text-[var(--color-ink)]">{account.dienstgeberId}</span>
						</div>
					{/if}
					<div class="flex items-center justify-between px-4 py-3">
						<span class="text-sm text-[var(--color-ink-muted)]">Documents</span>
						<span class="text-sm font-medium text-[var(--color-ink)]">{account.documentCount}</span>
					</div>
					<div class="flex items-center justify-between px-4 py-3">
						<span class="text-sm text-[var(--color-ink-muted)]">Last sync</span>
						<span class="text-sm text-[var(--color-ink)]">
							{account.lastSync ? formatDateTime(account.lastSync) : 'Never'}
						</span>
					</div>
					<div class="flex items-center justify-between px-4 py-3">
						<span class="text-sm text-[var(--color-ink-muted)]">Tags</span>
						<div class="flex flex-wrap gap-1">
							{#if account.tags.length > 0}
								{#each account.tags as tag}
									<span class="px-2 py-0.5 text-xs rounded-full bg-[var(--color-paper-inset)] text-[var(--color-ink-secondary)]">
										{tag}
									</span>
								{/each}
							{:else}
								<span class="text-sm text-[var(--color-ink-muted)]">—</span>
							{/if}
						</div>
					</div>
					<div class="flex items-center justify-between px-4 py-3">
						<span class="text-sm text-[var(--color-ink-muted)]">Created</span>
						<span class="text-sm text-[var(--color-ink)]">{formatDate(account.createdAt)}</span>
					</div>
				</div>
			{/if}
		</Card>

		<!-- Error message -->
		{#if account.status === 'error' && account.lastError}
			<div class="flex gap-3 p-4 rounded-lg bg-[var(--color-error-muted)]">
				<svg class="w-5 h-5 text-[var(--color-error)] flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="10"/>
					<path d="M12 8v4M12 16h.01"/>
				</svg>
				<div class="text-sm text-[var(--color-error)]">
					<p class="font-medium">Connection error</p>
					<p class="mt-0.5 opacity-80">{account.lastError}</p>
				</div>
			</div>
		{/if}

		<!-- Danger zone -->
		<Card padding="none">
			<div class="p-4 border-b border-black/6">
				<h2 class="font-semibold text-[var(--color-error)]">Danger Zone</h2>
			</div>
			<div class="p-4">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm font-medium text-[var(--color-ink)]">Delete this account</p>
						<p class="text-sm text-[var(--color-ink-muted)]">This will permanently delete the account and all its documents.</p>
					</div>
					<Button variant="danger" onclick={() => { showDeleteConfirm = true; }}>
						Delete account
					</Button>
				</div>
			</div>
		</Card>
	</div>

	<!-- Delete confirmation dialog -->
	{#if showDeleteConfirm}
		<div class="fixed inset-0 bg-black/50 backdrop-blur-sm z-[var(--z-modal)] flex items-center justify-center p-4">
			<div class="bg-[var(--color-paper-elevated)] rounded-xl shadow-2xl max-w-md w-full p-6 animate-in">
				<div class="flex items-center gap-3 mb-4">
					<div class="w-10 h-10 rounded-full bg-[var(--color-error-muted)] flex items-center justify-center">
						<svg class="w-5 h-5 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M3 6h18M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-[var(--color-ink)]">Delete account</h3>
						<p class="text-sm text-[var(--color-ink-muted)]">Are you sure?</p>
					</div>
				</div>
				<p class="text-sm text-[var(--color-ink-secondary)] mb-6">
					This will permanently delete <strong>{account.name}</strong> and all {account.documentCount} associated documents. This action cannot be undone.
				</p>
				<div class="flex justify-end gap-2">
					<Button variant="secondary" onclick={() => { showDeleteConfirm = false; }}>Cancel</Button>
					<Button variant="danger" onclick={handleDelete} loading={isDeleting}>Delete</Button>
				</div>
			</div>
		</div>
	{/if}
{/if}
