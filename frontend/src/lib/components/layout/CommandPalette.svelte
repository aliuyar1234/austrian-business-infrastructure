<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';

	interface SearchResult {
		id: string;
		type: 'account' | 'document' | 'action' | 'page';
		title: string;
		subtitle?: string;
		icon?: string;
		href?: string;
		action?: () => void;
	}

	let isOpen = $state(false);
	let query = $state('');
	let selectedIndex = $state(0);
	let inputRef = $state<HTMLInputElement | null>(null);

	// Navigation items
	const pages: SearchResult[] = [
		{ id: 'dashboard', type: 'page', title: 'Dashboard', subtitle: 'Overview', href: '/' },
		{ id: 'accounts', type: 'page', title: 'Accounts', subtitle: 'Manage accounts', href: '/accounts' },
		{ id: 'documents', type: 'page', title: 'Documents', subtitle: 'Databox documents', href: '/documents' },
		{ id: 'uva', type: 'page', title: 'UVA', subtitle: 'VAT returns', href: '/uva' },
		{ id: 'invoices', type: 'page', title: 'Invoices', subtitle: 'E-Rechnung', href: '/invoices' },
		{ id: 'sepa', type: 'page', title: 'SEPA', subtitle: 'Payments', href: '/sepa' },
		{ id: 'firmenbuch', type: 'page', title: 'Firmenbuch', subtitle: 'Company search', href: '/firmenbuch' },
		{ id: 'calendar', type: 'page', title: 'Calendar', subtitle: 'Deadlines', href: '/calendar' },
		{ id: 'team', type: 'page', title: 'Team', subtitle: 'Manage team', href: '/team' },
		{ id: 'settings', type: 'page', title: 'Settings', subtitle: 'Preferences', href: '/settings' },
	];

	// Quick actions
	const actions: SearchResult[] = [
		{ id: 'new-account', type: 'action', title: 'Add new account', subtitle: 'Connect FinanzOnline', href: '/accounts/new' },
		{ id: 'new-uva', type: 'action', title: 'Create UVA', subtitle: 'New VAT return', href: '/uva/new' },
		{ id: 'new-invoice', type: 'action', title: 'Create invoice', subtitle: 'New E-Rechnung', href: '/invoices/new' },
		{ id: 'sync-all', type: 'action', title: 'Sync all accounts', subtitle: 'Refresh databox', action: () => alert('Syncing...') },
	];

	// Mock accounts and documents for search
	const mockAccounts: SearchResult[] = [
		{ id: 'acc-1', type: 'account', title: 'Muster GmbH', subtitle: 'FinanzOnline • Active', href: '/accounts/1' },
		{ id: 'acc-2', type: 'account', title: 'Test AG', subtitle: 'FinanzOnline • Active', href: '/accounts/2' },
		{ id: 'acc-3', type: 'account', title: 'Demo GmbH', subtitle: 'ELDA • Error', href: '/accounts/3' },
	];

	const mockDocuments: SearchResult[] = [
		{ id: 'doc-1', type: 'document', title: 'Bescheid Umsatzsteuer 2024', subtitle: 'Muster GmbH • Today', href: '/documents/1' },
		{ id: 'doc-2', type: 'document', title: 'Ergänzungsersuchen', subtitle: 'Test AG • Yesterday', href: '/documents/2' },
		{ id: 'doc-3', type: 'document', title: 'Vorauszahlungsbescheid Q1', subtitle: 'Muster GmbH • 2 days ago', href: '/documents/3' },
	];

	let results = $derived(() => {
		if (!query.trim()) {
			return [...actions.slice(0, 3), ...pages.slice(0, 4)];
		}

		const q = query.toLowerCase();
		const filtered: SearchResult[] = [];

		// Search accounts
		mockAccounts.forEach(acc => {
			if (acc.title.toLowerCase().includes(q) || acc.subtitle?.toLowerCase().includes(q)) {
				filtered.push(acc);
			}
		});

		// Search documents
		mockDocuments.forEach(doc => {
			if (doc.title.toLowerCase().includes(q) || doc.subtitle?.toLowerCase().includes(q)) {
				filtered.push(doc);
			}
		});

		// Search actions
		actions.forEach(action => {
			if (action.title.toLowerCase().includes(q)) {
				filtered.push(action);
			}
		});

		// Search pages
		pages.forEach(page => {
			if (page.title.toLowerCase().includes(q) || page.subtitle?.toLowerCase().includes(q)) {
				filtered.push(page);
			}
		});

		return filtered.slice(0, 10);
	});

	function open() {
		isOpen = true;
		query = '';
		selectedIndex = 0;
		setTimeout(() => inputRef?.focus(), 50);
	}

	function close() {
		isOpen = false;
		query = '';
	}

	function selectResult(result: SearchResult) {
		if (result.action) {
			result.action();
		} else if (result.href) {
			goto(result.href);
		}
		close();
	}

	function handleKeyDown(e: KeyboardEvent) {
		const res = results();

		switch (e.key) {
			case 'ArrowDown':
				e.preventDefault();
				selectedIndex = Math.min(selectedIndex + 1, res.length - 1);
				break;
			case 'ArrowUp':
				e.preventDefault();
				selectedIndex = Math.max(selectedIndex - 1, 0);
				break;
			case 'Enter':
				e.preventDefault();
				if (res[selectedIndex]) {
					selectResult(res[selectedIndex]);
				}
				break;
			case 'Escape':
				e.preventDefault();
				close();
				break;
		}
	}

	function handleGlobalKeyDown(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
			e.preventDefault();
			if (isOpen) {
				close();
			} else {
				open();
			}
		}
	}

	function getTypeIcon(type: SearchResult['type']): string {
		switch (type) {
			case 'account':
				return '<rect width="16" height="20" x="4" y="2" rx="2" ry="2"/><path d="M9 22v-4h6v4"/><path d="M8 6h.01"/><path d="M16 6h.01"/><path d="M12 6h.01"/><path d="M12 10h.01"/><path d="M12 14h.01"/><path d="M16 10h.01"/><path d="M16 14h.01"/><path d="M8 10h.01"/><path d="M8 14h.01"/>';
			case 'document':
				return '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>';
			case 'action':
				return '<circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/>';
			case 'page':
				return '<rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><line x1="3" x2="21" y1="9" y2="9"/><line x1="9" x2="9" y1="21" y2="9"/>';
		}
	}

	function getTypeLabel(type: SearchResult['type']): string {
		switch (type) {
			case 'account': return 'Account';
			case 'document': return 'Document';
			case 'action': return 'Action';
			case 'page': return 'Page';
		}
	}

	onMount(() => {
		window.addEventListener('keydown', handleGlobalKeyDown);
	});

	onDestroy(() => {
		window.removeEventListener('keydown', handleGlobalKeyDown);
	});

	// Expose open function for external use
	export { open };
</script>

{#if isOpen}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 bg-black/50 backdrop-blur-sm z-[var(--z-modal)]"
		onclick={close}
		onkeydown={(e) => e.key === 'Escape' && close()}
		role="button"
		tabindex="-1"
	></div>

	<!-- Dialog -->
	<div class="fixed inset-x-4 top-[15vh] sm:inset-x-auto sm:left-1/2 sm:-translate-x-1/2 sm:w-full sm:max-w-xl z-[var(--z-modal)] animate-in">
		<div class="bg-[var(--color-paper-elevated)] rounded-xl shadow-2xl border border-black/10 overflow-hidden">
			<!-- Search input -->
			<div class="flex items-center gap-3 px-4 border-b border-black/6">
				<svg class="w-5 h-5 text-[var(--color-ink-muted)] flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="11" cy="11" r="8"/>
					<path d="m21 21-4.3-4.3"/>
				</svg>
				<input
					bind:this={inputRef}
					bind:value={query}
					onkeydown={handleKeyDown}
					type="text"
					placeholder="Search accounts, documents, actions..."
					class="flex-1 h-14 bg-transparent text-[var(--color-ink)] placeholder:text-[var(--color-ink-muted)] text-base focus:outline-none"
				/>
				<kbd class="hidden sm:flex items-center gap-1 px-2 py-1 rounded bg-black/5 text-xs text-[var(--color-ink-muted)]">
					<span class="text-sm">⌘</span>K
				</kbd>
			</div>

			<!-- Results -->
			<div class="max-h-80 overflow-y-auto p-2">
				{#if results().length === 0}
					<div class="py-8 text-center text-[var(--color-ink-muted)]">
						<svg class="w-10 h-10 mx-auto mb-3 opacity-40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
							<circle cx="11" cy="11" r="8"/>
							<path d="m21 21-4.3-4.3"/>
						</svg>
						<p class="text-sm">No results found for "{query}"</p>
					</div>
				{:else}
					{#each results() as result, i}
						<button
							onclick={() => selectResult(result)}
							class={`
								w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left
								transition-colors duration-75
								${i === selectedIndex ? 'bg-[var(--color-accent-muted)] text-[var(--color-accent)]' : 'hover:bg-[var(--color-paper-inset)]'}
							`}
							onmouseenter={() => { selectedIndex = i; }}
						>
							<div class={`
								w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0
								${i === selectedIndex ? 'bg-[var(--color-accent)]/10' : 'bg-[var(--color-paper-inset)]'}
							`}>
								<svg
									class={`w-4 h-4 ${i === selectedIndex ? 'text-[var(--color-accent)]' : 'text-[var(--color-ink-muted)]'}`}
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
								>
									{@html getTypeIcon(result.type)}
								</svg>
							</div>
							<div class="flex-1 min-w-0">
								<p class={`text-sm font-medium truncate ${i === selectedIndex ? 'text-[var(--color-accent)]' : 'text-[var(--color-ink)]'}`}>
									{result.title}
								</p>
								{#if result.subtitle}
									<p class="text-xs text-[var(--color-ink-muted)] truncate">{result.subtitle}</p>
								{/if}
							</div>
							<span class={`
								text-[10px] font-medium uppercase tracking-wide px-1.5 py-0.5 rounded
								${i === selectedIndex ? 'bg-[var(--color-accent)]/10 text-[var(--color-accent)]' : 'bg-black/5 text-[var(--color-ink-muted)]'}
							`}>
								{getTypeLabel(result.type)}
							</span>
						</button>
					{/each}
				{/if}
			</div>

			<!-- Footer -->
			<div class="px-4 py-2.5 border-t border-black/6 flex items-center justify-between text-xs text-[var(--color-ink-muted)]">
				<div class="flex items-center gap-4">
					<span class="flex items-center gap-1">
						<kbd class="px-1.5 py-0.5 rounded bg-black/5">↑</kbd>
						<kbd class="px-1.5 py-0.5 rounded bg-black/5">↓</kbd>
						to navigate
					</span>
					<span class="flex items-center gap-1">
						<kbd class="px-1.5 py-0.5 rounded bg-black/5">↵</kbd>
						to select
					</span>
				</div>
				<span class="flex items-center gap-1">
					<kbd class="px-1.5 py-0.5 rounded bg-black/5">esc</kbd>
					to close
				</span>
			</div>
		</div>
	</div>
{/if}
