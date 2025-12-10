<script lang="ts">
	import { goto } from '$app/navigation';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	interface Signer {
		id: string;
		name: string;
		email: string;
		reason?: string;
	}

	let title = $state('');
	let selectedFile = $state<File | null>(null);
	let signers = $state<Signer[]>([{ id: crypto.randomUUID(), name: '', email: '', reason: '' }]);
	let expiryDays = $state(14);
	let message = $state('');
	let isSubmitting = $state(false);

	function addSigner() {
		signers = [...signers, { id: crypto.randomUUID(), name: '', email: '', reason: '' }];
	}

	function removeSigner(id: string) {
		if (signers.length > 1) {
			signers = signers.filter(s => s.id !== id);
		}
	}

	function moveSigner(index: number, direction: 'up' | 'down') {
		const newIndex = direction === 'up' ? index - 1 : index + 1;
		if (newIndex < 0 || newIndex >= signers.length) return;

		const newSigners = [...signers];
		[newSigners[index], newSigners[newIndex]] = [newSigners[newIndex], newSigners[index]];
		signers = newSigners;
	}

	function handleFileSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files && input.files[0]) {
			selectedFile = input.files[0];
			if (!title) {
				title = input.files[0].name.replace(/\.pdf$/i, '');
			}
		}
	}

	function handleFileDrop(event: DragEvent) {
		event.preventDefault();
		if (event.dataTransfer?.files && event.dataTransfer.files[0]) {
			const file = event.dataTransfer.files[0];
			if (file.type === 'application/pdf') {
				selectedFile = file;
				if (!title) {
					title = file.name.replace(/\.pdf$/i, '');
				}
			}
		}
	}

	let isValid = $derived(
		title.trim() !== '' &&
		selectedFile !== null &&
		signers.every(s => s.name.trim() !== '' && s.email.trim() !== '' && s.email.includes('@'))
	);

	async function handleSubmit() {
		if (!isValid || isSubmitting) return;

		isSubmitting = true;
		try {
			// TODO: Call API to create signature request
			const formData = new FormData();
			formData.append('title', title);
			formData.append('file', selectedFile!);
			formData.append('signers', JSON.stringify(signers.map((s, i) => ({
				name: s.name,
				email: s.email,
				reason: s.reason,
				order_index: i
			}))));
			formData.append('expires_in_days', expiryDays.toString());
			if (message) {
				formData.append('message', message);
			}

			// const response = await fetch('/api/v1/signatures', {
			// 	method: 'POST',
			// 	body: formData
			// });

			// Simulate success
			await new Promise(resolve => setTimeout(resolve, 1000));

			goto('/signatures');
		} catch (error) {
			console.error('Failed to create signature request:', error);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="max-w-3xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<Button variant="ghost" href="/signatures">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m15 18-6-6 6-6"/>
			</svg>
		</Button>
		<div>
			<h1 class="text-2xl font-semibold text-[var(--color-ink)]">Neue Signaturanfrage</h1>
			<p class="text-[var(--color-ink-muted)]">Dokument zur qualifizierten Signatur freigeben</p>
		</div>
	</div>

	<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-6">
		<!-- Document upload -->
		<Card>
			<h2 class="text-lg font-medium text-[var(--color-ink)] mb-4">Dokument</h2>

			{#if selectedFile}
				<div class="flex items-center gap-4 p-4 bg-[var(--color-paper-inset)] rounded-lg">
					<div class="w-12 h-12 rounded-lg bg-[var(--color-accent-muted)] flex items-center justify-center">
						<svg class="w-6 h-6 text-[var(--color-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
							<polyline points="14 2 14 8 20 8"/>
						</svg>
					</div>
					<div class="flex-1 min-w-0">
						<p class="font-medium text-[var(--color-ink)] truncate">{selectedFile.name}</p>
						<p class="text-sm text-[var(--color-ink-muted)]">{(selectedFile.size / 1024 / 1024).toFixed(2)} MB</p>
					</div>
					<Button variant="ghost" size="sm" onclick={() => selectedFile = null}>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<line x1="18" x2="6" y1="6" y2="18"/>
							<line x1="6" x2="18" y1="6" y2="18"/>
						</svg>
					</Button>
				</div>
			{:else}
				<div
					role="button"
					tabindex="0"
					class="border-2 border-dashed border-[var(--color-border)] rounded-lg p-8 text-center hover:border-[var(--color-accent)] transition-colors cursor-pointer"
					ondrop={handleFileDrop}
					ondragover={(e) => e.preventDefault()}
					onclick={() => document.getElementById('file-input')?.click()}
					onkeydown={(e) => e.key === 'Enter' && document.getElementById('file-input')?.click()}
				>
					<svg class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
						<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
						<polyline points="17 8 12 3 7 8"/>
						<line x1="12" x2="12" y1="3" y2="15"/>
					</svg>
					<p class="mt-4 text-[var(--color-ink)]">PDF-Datei hier ablegen oder klicken</p>
					<p class="mt-1 text-sm text-[var(--color-ink-muted)]">Maximale Dateigroesse: 100 MB</p>
				</div>
				<input
					id="file-input"
					type="file"
					accept=".pdf,application/pdf"
					class="hidden"
					onchange={handleFileSelect}
				/>
			{/if}

			<div class="mt-4">
				<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Titel</label>
				<Input bind:value={title} placeholder="Beschreibender Titel fuer die Signaturanfrage" />
			</div>
		</Card>

		<!-- Signers -->
		<Card>
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-medium text-[var(--color-ink)]">Unterzeichner</h2>
				<Button variant="secondary" size="sm" onclick={addSigner}>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="12" x2="12" y1="5" y2="19"/>
						<line x1="5" x2="19" y1="12" y2="12"/>
					</svg>
					Hinzufuegen
				</Button>
			</div>

			<p class="text-sm text-[var(--color-ink-muted)] mb-4">
				Die Unterzeichner werden in der angegebenen Reihenfolge zur Signatur aufgefordert.
			</p>

			<div class="space-y-4">
				{#each signers as signer, index (signer.id)}
					<div class="p-4 bg-[var(--color-paper-inset)] rounded-lg">
						<div class="flex items-start gap-4">
							<div class="flex flex-col gap-1">
								<button
									type="button"
									class="p-1 rounded hover:bg-[var(--color-paper)] disabled:opacity-30"
									disabled={index === 0}
									onclick={() => moveSigner(index, 'up')}
								>
									<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<path d="m18 15-6-6-6 6"/>
									</svg>
								</button>
								<span class="text-center text-sm font-medium text-[var(--color-ink-muted)]">{index + 1}</span>
								<button
									type="button"
									class="p-1 rounded hover:bg-[var(--color-paper)] disabled:opacity-30"
									disabled={index === signers.length - 1}
									onclick={() => moveSigner(index, 'down')}
								>
									<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<path d="m6 9 6 6 6-6"/>
									</svg>
								</button>
							</div>
							<div class="flex-1 grid gap-4 sm:grid-cols-2">
								<div>
									<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Name</label>
									<Input bind:value={signer.name} placeholder="Vollstaendiger Name" required />
								</div>
								<div>
									<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">E-Mail</label>
									<Input type="email" bind:value={signer.email} placeholder="email@example.com" required />
								</div>
								<div class="sm:col-span-2">
									<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Signaturgrund (optional)</label>
									<Input bind:value={signer.reason} placeholder="z.B. Genehmigung, Kenntnisnahme" />
								</div>
							</div>
							{#if signers.length > 1}
								<button
									type="button"
									class="p-2 rounded hover:bg-[var(--color-paper)]"
									onclick={() => removeSigner(signer.id)}
								>
									<svg class="w-4 h-4 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<line x1="18" x2="6" y1="6" y2="18"/>
										<line x1="6" x2="18" y1="6" y2="18"/>
									</svg>
								</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</Card>

		<!-- Settings -->
		<Card>
			<h2 class="text-lg font-medium text-[var(--color-ink)] mb-4">Einstellungen</h2>

			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Gueltigkeitsdauer</label>
					<select bind:value={expiryDays} class="input h-10 w-full">
						<option value={7}>7 Tage</option>
						<option value={14}>14 Tage</option>
						<option value={30}>30 Tage</option>
						<option value={60}>60 Tage</option>
						<option value={90}>90 Tage</option>
					</select>
				</div>
			</div>

			<div class="mt-4">
				<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Nachricht an Unterzeichner (optional)</label>
				<textarea
					bind:value={message}
					class="input w-full min-h-[100px]"
					placeholder="Diese Nachricht wird den Unterzeichnern in der E-Mail-Einladung angezeigt..."
				></textarea>
			</div>
		</Card>

		<!-- Actions -->
		<div class="flex justify-end gap-3">
			<Button variant="secondary" href="/signatures">Abbrechen</Button>
			<Button type="submit" disabled={!isValid || isSubmitting}>
				{#if isSubmitting}
					<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 12a9 9 0 1 1-6.219-8.56"/>
					</svg>
					Wird erstellt...
				{:else}
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
					</svg>
					Signaturanfrage erstellen
				{/if}
			</Button>
		</div>
	</form>
</div>
