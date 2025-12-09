<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { formatDate, formatDateTime } from '$lib/utils';
	import { api } from '$lib/api/client';
	import { toast } from '$lib/stores/toast';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	type DocumentType = 'bescheid' | 'ersuchen' | 'info' | 'quittung' | 'other';
	type DocumentStatus = 'unread' | 'read' | 'archived';

	interface Document {
		id: string;
		title: string;
		type: DocumentType;
		status: DocumentStatus;
		accountId: string;
		accountName: string;
		receivedAt: Date;
		hasDeadline: boolean;
		deadline?: Date;
		fileSize: number;
		mimeType: string;
		content?: string;
	}

	let document = $state<Document | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let pdfUrl = $state<string | null>(null);
	let showSidebar = $state(true);

	onMount(async () => {
		await loadDocument();
	});

	async function loadDocument() {
		loading = true;
		error = null;

		try {
			// Mock data for demo
			document = {
				id: $page.params.id,
				title: 'Bescheid Umsatzsteuer 2024',
				type: 'bescheid',
				status: 'read',
				accountId: '1',
				accountName: 'Muster GmbH',
				receivedAt: new Date(),
				hasDeadline: false,
				fileSize: 245678,
				mimeType: 'application/pdf',
			};

			// Mark as read
			if (document.status === 'unread') {
				await markAsRead();
			}

			// Build PDF URL for viewer
			pdfUrl = `/api/v1/documents/${$page.params.id}/content`;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load document';
		} finally {
			loading = false;
		}
	}

	async function markAsRead() {
		try {
			await api.post(`/api/v1/documents/${$page.params.id}/read`);
			if (document) {
				document = { ...document, status: 'read' };
			}
		} catch {
			// Silently fail - not critical
		}
	}

	async function downloadDocument() {
		if (!document) return;
		try {
			await api.download(`/api/v1/documents/${$page.params.id}/content`, `${document.title}.pdf`);
			toast.success('Download started', 'Your document is being downloaded');
		} catch {
			toast.error('Download failed', 'Could not download the document');
		}
	}

	async function archiveDocument() {
		if (!document) return;
		try {
			await api.post(`/api/v1/documents/${$page.params.id}/archive`);
			document = { ...document, status: 'archived' };
			toast.success('Document archived', 'The document has been archived');
		} catch {
			toast.error('Archive failed', 'Could not archive the document');
		}
	}

	function getTypeLabel(type: DocumentType): string {
		switch (type) {
			case 'bescheid': return 'Bescheid';
			case 'ersuchen': return 'Ersuchen';
			case 'info': return 'Information';
			case 'quittung': return 'Quittung';
			case 'other': return 'Sonstige';
		}
	}

	function getTypeVariant(type: DocumentType): 'default' | 'warning' | 'info' | 'success' {
		switch (type) {
			case 'bescheid': return 'default';
			case 'ersuchen': return 'warning';
			case 'info': return 'info';
			case 'quittung': return 'success';
			default: return 'default';
		}
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

<svelte:head>
	<title>{document?.title ?? 'Document'} - Austrian Business Infrastructure</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center h-[calc(100vh-10rem)]">
		<div class="flex flex-col items-center gap-4">
			<div class="w-10 h-10 rounded-lg bg-[var(--color-accent)] flex items-center justify-center animate-pulse">
				<span class="text-white font-bold text-sm">ABI</span>
			</div>
			<p class="text-sm text-[var(--color-ink-muted)]">Loading document...</p>
		</div>
	</div>
{:else if error}
	<div class="max-w-4xl mx-auto animate-in">
		<Card class="text-center py-12">
			<svg class="w-12 h-12 mx-auto text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<circle cx="12" cy="12" r="10"/>
				<path d="M12 8v4M12 16h.01"/>
			</svg>
			<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">Failed to load document</h3>
			<p class="mt-2 text-[var(--color-ink-muted)]">{error}</p>
			<Button variant="secondary" href="/documents" class="mt-6">Back to documents</Button>
		</Card>
	</div>
{:else if document}
	<div class="flex h-[calc(100vh-5rem)] -m-6">
		<!-- PDF Viewer -->
		<div class="flex-1 flex flex-col bg-[var(--color-paper-inset)]">
			<!-- Toolbar -->
			<div class="h-14 bg-[var(--color-paper-elevated)] border-b border-black/6 flex items-center justify-between px-4">
				<div class="flex items-center gap-3">
					<a href="/documents" aria-label="Back to documents" class="w-9 h-9 flex items-center justify-center rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors">
						<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="m12 19-7-7 7-7M19 12H5"/>
						</svg>
					</a>
					<div class="h-6 w-px bg-black/10"></div>
					<span class="text-sm font-medium text-[var(--color-ink)] truncate max-w-md">{document.title}</span>
				</div>
				<div class="flex items-center gap-2">
					<Button variant="ghost" size="sm" onclick={downloadDocument}>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
							<polyline points="7 10 12 15 17 10"/>
							<line x1="12" x2="12" y1="15" y2="3"/>
						</svg>
						Download
					</Button>
					<Button variant="ghost" size="sm" onclick={() => { showSidebar = !showSidebar; }}>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<rect width="18" height="18" x="3" y="3" rx="2" ry="2"/>
							<line x1="15" x2="15" y1="3" y2="21"/>
						</svg>
					</Button>
				</div>
			</div>

			<!-- PDF embed / placeholder -->
			<div class="flex-1 flex items-center justify-center p-6">
				{#if pdfUrl}
					<!-- Native PDF embed -->
					<div class="w-full h-full bg-white rounded-lg shadow-lg overflow-hidden">
						<object
							data={pdfUrl}
							type="application/pdf"
							title="PDF document viewer"
							class="w-full h-full"
						>
							<!-- Fallback for browsers without PDF support -->
							<div class="w-full h-full flex flex-col items-center justify-center p-8 text-center">
								<svg class="w-16 h-16 text-[var(--color-ink-muted)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
									<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
									<polyline points="14 2 14 8 20 8"/>
									<line x1="16" x2="8" y1="13" y2="13"/>
									<line x1="16" x2="8" y1="17" y2="17"/>
									<line x1="10" x2="8" y1="9" y2="9"/>
								</svg>
								<h3 class="text-lg font-medium text-[var(--color-ink)]">PDF Preview</h3>
								<p class="text-sm text-[var(--color-ink-muted)] mt-2 max-w-sm">
									Your browser doesn't support embedded PDFs. Click the button below to download and view the document.
								</p>
								<Button onclick={downloadDocument} class="mt-4">
									<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
										<polyline points="7 10 12 15 17 10"/>
										<line x1="12" x2="12" y1="15" y2="3"/>
									</svg>
									Download PDF
								</Button>
							</div>
						</object>
					</div>
				{:else}
					<div class="text-center">
						<svg class="w-16 h-16 mx-auto text-[var(--color-ink-muted)] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
							<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
							<polyline points="14 2 14 8 20 8"/>
						</svg>
						<p class="text-[var(--color-ink-muted)]">No preview available</p>
					</div>
				{/if}
			</div>
		</div>

		<!-- Sidebar -->
		{#if showSidebar}
			<div class="w-80 bg-[var(--color-paper-elevated)] border-l border-black/6 flex flex-col">
				<!-- Document info -->
				<div class="p-4 border-b border-black/6">
					<h2 class="font-semibold text-[var(--color-ink)]">Document Details</h2>
				</div>

				<div class="flex-1 overflow-y-auto">
					<div class="divide-y divide-black/4">
						<div class="px-4 py-3">
							<span class="text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wide">Type</span>
							<div class="mt-1">
								<Badge variant={getTypeVariant(document.type)}>
									{getTypeLabel(document.type)}
								</Badge>
							</div>
						</div>

						<div class="px-4 py-3">
							<span class="text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wide">Account</span>
							<p class="mt-1 text-sm text-[var(--color-ink)]">{document.accountName}</p>
						</div>

						<div class="px-4 py-3">
							<span class="text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wide">Received</span>
							<p class="mt-1 text-sm text-[var(--color-ink)]">{formatDateTime(document.receivedAt)}</p>
						</div>

						{#if document.hasDeadline && document.deadline}
							<div class="px-4 py-3">
								<span class="text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wide">Deadline</span>
								<div class="mt-1 flex items-center gap-2">
									<Badge variant="warning" dot>{formatDate(document.deadline)}</Badge>
								</div>
							</div>
						{/if}

						<div class="px-4 py-3">
							<span class="text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wide">File Size</span>
							<p class="mt-1 text-sm text-[var(--color-ink)]">{formatFileSize(document.fileSize)}</p>
						</div>

						<div class="px-4 py-3">
							<span class="text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wide">Status</span>
							<p class="mt-1 text-sm text-[var(--color-ink)] capitalize">{document.status}</p>
						</div>
					</div>
				</div>

				<!-- Actions -->
				<div class="p-4 border-t border-black/6 space-y-2">
					<Button variant="secondary" class="w-full" onclick={downloadDocument}>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
							<polyline points="7 10 12 15 17 10"/>
							<line x1="12" x2="12" y1="15" y2="3"/>
						</svg>
						Download
					</Button>
					{#if document.status !== 'archived'}
						<Button variant="ghost" class="w-full" onclick={archiveDocument}>
							<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<rect width="20" height="5" x="2" y="3" rx="1"/>
								<path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8"/>
								<path d="M10 12h4"/>
							</svg>
							Archive
						</Button>
					{/if}
				</div>
			</div>
		{/if}
	</div>
{/if}
