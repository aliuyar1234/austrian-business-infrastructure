<script lang="ts">
	import { goto } from '$app/navigation';

	interface Props {
		class?: string;
	}

	let { class: className = '' }: Props = $props();

	let showHelp = $state(false);

	const shortcuts: { key: string; description: string; action: () => void }[] = [
		{ key: '⌘K', description: 'Open command palette', action: () => {} },
		{ key: 'g h', description: 'Go to Dashboard', action: () => goto('/') },
		{ key: 'g a', description: 'Go to Accounts', action: () => goto('/accounts') },
		{ key: 'g d', description: 'Go to Documents', action: () => goto('/documents') },
		{ key: 'g u', description: 'Go to UVA', action: () => goto('/uva') },
		{ key: 'g i', description: 'Go to Invoices', action: () => goto('/invoices') },
		{ key: 'g s', description: 'Go to Settings', action: () => goto('/settings') },
		{ key: '?', description: 'Show keyboard shortcuts', action: () => { showHelp = true; } },
		{ key: 'Esc', description: 'Close dialogs', action: () => { showHelp = false; } },
	];

	let pendingKey = $state<string | null>(null);
	let pendingTimeout: ReturnType<typeof setTimeout> | null = null;

	function handleKeyDown(e: KeyboardEvent) {
		// Don't trigger shortcuts when typing in input fields
		const target = e.target as HTMLElement;
		if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
			return;
		}

		// Handle Cmd/Ctrl combinations
		if (e.metaKey || e.ctrlKey) {
			return; // Let CommandPalette handle these
		}

		// Handle ? for help
		if (e.key === '?' && !e.shiftKey) {
			e.preventDefault();
			showHelp = true;
			return;
		}

		// Handle Escape
		if (e.key === 'Escape') {
			showHelp = false;
			pendingKey = null;
			if (pendingTimeout) clearTimeout(pendingTimeout);
			return;
		}

		// Handle g + <letter> combinations (vim-style navigation)
		if (pendingKey === 'g') {
			e.preventDefault();
			pendingKey = null;
			if (pendingTimeout) clearTimeout(pendingTimeout);

			switch (e.key.toLowerCase()) {
				case 'h': goto('/'); break;
				case 'a': goto('/accounts'); break;
				case 'd': goto('/documents'); break;
				case 'u': goto('/uva'); break;
				case 'i': goto('/invoices'); break;
				case 's': goto('/settings'); break;
				case 'c': goto('/calendar'); break;
				case 't': goto('/team'); break;
				case 'f': goto('/firmenbuch'); break;
				case 'p': goto('/sepa'); break;
			}
			return;
		}

		// Start pending for g key
		if (e.key === 'g') {
			e.preventDefault();
			pendingKey = 'g';
			if (pendingTimeout) clearTimeout(pendingTimeout);
			pendingTimeout = setTimeout(() => {
				pendingKey = null;
			}, 1000);
			return;
		}
	}
</script>

<svelte:window onkeydown={handleKeyDown} />

<!-- Pending key indicator -->
{#if pendingKey}
	<div class="fixed bottom-4 right-4 z-[var(--z-toast)] animate-in">
		<div class="bg-[var(--color-paper-elevated)] border border-black/10 rounded-lg shadow-lg px-3 py-2">
			<span class="text-sm text-[var(--color-ink-muted)]">
				<kbd class="px-1.5 py-0.5 bg-[var(--color-paper-inset)] rounded text-xs font-mono">{pendingKey}</kbd>
				<span class="ml-1">waiting for next key...</span>
			</span>
		</div>
	</div>
{/if}

<!-- Help modal -->
{#if showHelp}
	<div class="fixed inset-0 bg-black/50 backdrop-blur-sm z-[var(--z-modal)] flex items-center justify-center p-4">
		<div class="bg-[var(--color-paper-elevated)] rounded-xl shadow-2xl max-w-md w-full max-h-[90vh] overflow-hidden animate-in {className}">
			<div class="p-4 border-b border-black/6 flex items-center justify-between">
				<h2 class="text-lg font-semibold text-[var(--color-ink)]">Keyboard Shortcuts</h2>
				<button onclick={() => { showHelp = false; }} class="p-2 rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors" aria-label="Close">
					<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M18 6 6 18M6 6l12 12"/>
					</svg>
				</button>
			</div>

			<div class="p-4 overflow-y-auto max-h-[60vh]">
				<div class="space-y-4">
					<!-- General -->
					<div>
						<h3 class="text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wide mb-2">General</h3>
						<div class="space-y-1">
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Open command palette</span>
								<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">⌘K</kbd>
							</div>
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Show shortcuts</span>
								<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">?</kbd>
							</div>
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Close dialogs</span>
								<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">Esc</kbd>
							</div>
						</div>
					</div>

					<!-- Navigation -->
					<div>
						<h3 class="text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wide mb-2">Navigation</h3>
						<p class="text-xs text-[var(--color-ink-muted)] mb-2">Press <kbd class="px-1 py-0.5 bg-[var(--color-paper-inset)] rounded">g</kbd> then a letter</p>
						<div class="space-y-1">
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Dashboard</span>
								<div class="flex gap-1">
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">g</kbd>
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">h</kbd>
								</div>
							</div>
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Accounts</span>
								<div class="flex gap-1">
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">g</kbd>
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">a</kbd>
								</div>
							</div>
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Documents</span>
								<div class="flex gap-1">
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">g</kbd>
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">d</kbd>
								</div>
							</div>
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">UVA</span>
								<div class="flex gap-1">
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">g</kbd>
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">u</kbd>
								</div>
							</div>
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Invoices</span>
								<div class="flex gap-1">
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">g</kbd>
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">i</kbd>
								</div>
							</div>
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Calendar</span>
								<div class="flex gap-1">
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">g</kbd>
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">c</kbd>
								</div>
							</div>
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-[var(--color-ink-secondary)]">Settings</span>
								<div class="flex gap-1">
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">g</kbd>
									<kbd class="px-2 py-1 bg-[var(--color-paper-inset)] rounded text-xs font-mono text-[var(--color-ink-muted)]">s</kbd>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}
