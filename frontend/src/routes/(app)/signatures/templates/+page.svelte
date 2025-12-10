<script lang="ts">
	import { formatDate } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	interface SignerTemplate {
		role: string;
		name?: string;
		email?: string;
		reason?: string;
		orderIndex: number;
	}

	interface Template {
		id: string;
		name: string;
		description?: string;
		signers: SignerTemplate[];
		usageCount: number;
		isActive: boolean;
		createdAt: Date;
		updatedAt: Date;
	}

	// Mock templates
	let templates = $state<Template[]>([
		{
			id: '1',
			name: 'Arbeitsvertrag Standard',
			description: 'Standard-Arbeitsvertrag mit Mitarbeiter und Geschaeftsfuehrer',
			signers: [
				{ role: 'Mitarbeiter', orderIndex: 1 },
				{ role: 'Geschaeftsfuehrer', orderIndex: 2 }
			],
			usageCount: 45,
			isActive: true,
			createdAt: new Date(Date.now() - 30 * 86400000),
			updatedAt: new Date(Date.now() - 7 * 86400000)
		},
		{
			id: '2',
			name: 'Kundenvertrag',
			description: 'Vertrag mit Kunde und internem Vertreter',
			signers: [
				{ role: 'Interner Vertreter', orderIndex: 1 },
				{ role: 'Kunde', orderIndex: 2 }
			],
			usageCount: 32,
			isActive: true,
			createdAt: new Date(Date.now() - 60 * 86400000),
			updatedAt: new Date(Date.now() - 14 * 86400000)
		},
		{
			id: '3',
			name: 'Jahresabschluss 3-fach',
			description: 'Jahresabschluss mit Steuerberater, Geschaeftsfuehrer und Pruefer',
			signers: [
				{ role: 'Steuerberater', orderIndex: 1 },
				{ role: 'Geschaeftsfuehrer', orderIndex: 2 },
				{ role: 'Wirtschaftspruefer', orderIndex: 3 }
			],
			usageCount: 12,
			isActive: true,
			createdAt: new Date(Date.now() - 90 * 86400000),
			updatedAt: new Date(Date.now() - 30 * 86400000)
		},
		{
			id: '4',
			name: 'Lohnzettel (inaktiv)',
			description: 'Alte Vorlage fuer Lohnzettel',
			signers: [
				{ role: 'HR-Leitung', orderIndex: 1 }
			],
			usageCount: 156,
			isActive: false,
			createdAt: new Date(Date.now() - 180 * 86400000),
			updatedAt: new Date(Date.now() - 60 * 86400000)
		}
	]);

	let searchQuery = $state('');
	let showInactive = $state(false);
	let showCreateModal = $state(false);
	let editingTemplate = $state<Template | null>(null);

	let filteredTemplates = $derived(
		templates.filter(t => {
			const matchesSearch = !searchQuery ||
				t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				t.description?.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesActive = showInactive || t.isActive;
			return matchesSearch && matchesActive;
		})
	);

	function deleteTemplate(id: string) {
		if (confirm('Vorlage wirklich loeschen?')) {
			templates = templates.filter(t => t.id !== id);
		}
	}

	function toggleActive(id: string) {
		templates = templates.map(t =>
			t.id === id ? { ...t, isActive: !t.isActive } : t
		);
	}
</script>

<div class="max-w-5xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold text-[var(--color-ink)]">Signaturvorlagen</h1>
			<p class="text-[var(--color-ink-muted)]">
				{templates.filter(t => t.isActive).length} aktive Vorlagen
			</p>
		</div>
		<Button href="/signatures/templates/new">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<line x1="12" x2="12" y1="5" y2="19"/>
				<line x1="5" x2="19" y1="12" y2="12"/>
			</svg>
			Neue Vorlage
		</Button>
	</div>

	<!-- Filters -->
	<Card>
		<div class="flex flex-col sm:flex-row gap-4">
			<div class="flex-1">
				<Input
					type="search"
					bind:value={searchQuery}
					placeholder="Vorlagen suchen..."
				/>
			</div>
			<label class="flex items-center gap-2 text-sm text-[var(--color-ink-secondary)]">
				<input
					type="checkbox"
					bind:checked={showInactive}
					class="rounded border-[var(--color-border)]"
				/>
				Inaktive anzeigen
			</label>
		</div>
	</Card>

	<!-- Templates list -->
	{#if filteredTemplates.length === 0}
		<Card class="text-center py-12">
			<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
				<polyline points="14 2 14 8 20 8"/>
				<line x1="12" x2="12" y1="11" y2="17"/>
				<line x1="9" x2="15" y1="14" y2="14"/>
			</svg>
			<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">Keine Vorlagen gefunden</h3>
			<p class="mt-2 text-[var(--color-ink-muted)]">
				Erstellen Sie eine Vorlage fuer wiederkehrende Signaturanfragen.
			</p>
			<Button class="mt-4" href="/signatures/templates/new">
				Erste Vorlage erstellen
			</Button>
		</Card>
	{:else}
		<div class="space-y-4">
			{#each filteredTemplates as template}
				<Card class="{!template.isActive ? 'opacity-60' : ''}">
					<div class="flex items-start justify-between gap-4">
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2">
								<h3 class="font-medium text-[var(--color-ink)]">{template.name}</h3>
								{#if !template.isActive}
									<Badge variant="default" size="sm">Inaktiv</Badge>
								{/if}
							</div>
							{#if template.description}
								<p class="mt-1 text-sm text-[var(--color-ink-muted)]">{template.description}</p>
							{/if}

							<!-- Signers preview -->
							<div class="mt-3 flex flex-wrap gap-2">
								{#each template.signers as signer, i}
									<div class="flex items-center gap-1 px-2 py-1 bg-[var(--color-paper-inset)] rounded text-sm">
										<span class="text-[var(--color-ink-muted)]">{i + 1}.</span>
										<span class="text-[var(--color-ink)]">{signer.role}</span>
									</div>
								{/each}
							</div>

							<div class="mt-3 flex items-center gap-4 text-xs text-[var(--color-ink-muted)]">
								<span>{template.usageCount}x verwendet</span>
								<span>Erstellt {formatDate(template.createdAt)}</span>
							</div>
						</div>

						<div class="flex items-center gap-2">
							<Button variant="secondary" size="sm" href="/signatures/new?template={template.id}">
								Verwenden
							</Button>
							<Button variant="ghost" size="sm" href="/signatures/templates/{template.id}">
								<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
									<path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
								</svg>
							</Button>
							<button
								class="p-2 hover:bg-[var(--color-paper-inset)] rounded"
								onclick={() => toggleActive(template.id)}
								title={template.isActive ? 'Deaktivieren' : 'Aktivieren'}
							>
								<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									{#if template.isActive}
										<path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
										<line x1="1" x2="23" y1="1" y2="23"/>
									{:else}
										<path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/>
										<circle cx="12" cy="12" r="3"/>
									{/if}
								</svg>
							</button>
							<button
								class="p-2 hover:bg-[var(--color-paper-inset)] rounded"
								onclick={() => deleteTemplate(template.id)}
								title="Loeschen"
							>
								<svg class="w-4 h-4 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<polyline points="3 6 5 6 21 6"/>
									<path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
								</svg>
							</button>
						</div>
					</div>
				</Card>
			{/each}
		</div>
	{/if}
</div>
