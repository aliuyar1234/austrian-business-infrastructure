<script lang="ts">
	import { goto } from '$app/navigation';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	interface SignerTemplate {
		id: string;
		role: string;
		name: string;
		email: string;
		reason: string;
	}

	let name = $state('');
	let description = $state('');
	let signers = $state<SignerTemplate[]>([
		{ id: crypto.randomUUID(), role: '', name: '', email: '', reason: '' }
	]);
	let expiryDays = $state(14);
	let autoRemind = $state(true);
	let remindDays = $state(7);
	let isSubmitting = $state(false);

	function addSigner() {
		signers = [...signers, { id: crypto.randomUUID(), role: '', name: '', email: '', reason: '' }];
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

	let isValid = $derived(
		name.trim() !== '' &&
		signers.every(s => s.role.trim() !== '')
	);

	async function handleSubmit() {
		if (!isValid || isSubmitting) return;

		isSubmitting = true;
		try {
			// TODO: Call API
			// await fetch('/api/v1/signature-templates', {
			//   method: 'POST',
			//   headers: { 'Content-Type': 'application/json' },
			//   body: JSON.stringify({
			//     name,
			//     description,
			//     signers: signers.map((s, i) => ({
			//       role: s.role,
			//       name: s.name || null,
			//       email: s.email || null,
			//       reason: s.reason || null,
			//       order_index: i
			//     })),
			//     settings: {
			//       expiry_days: expiryDays,
			//       auto_remind: autoRemind,
			//       remind_days: remindDays
			//     }
			//   })
			// });

			await new Promise(resolve => setTimeout(resolve, 1000));
			goto('/signatures/templates');
		} catch (error) {
			console.error('Failed to create template:', error);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="max-w-3xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<Button variant="ghost" href="/signatures/templates">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m15 18-6-6 6-6"/>
			</svg>
		</Button>
		<div>
			<h1 class="text-2xl font-semibold text-[var(--color-ink)]">Neue Signaturvorlage</h1>
			<p class="text-[var(--color-ink-muted)]">Vorlage fuer wiederkehrende Signaturanfragen erstellen</p>
		</div>
	</div>

	<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-6">
		<!-- Basic info -->
		<Card>
			<h2 class="text-lg font-medium text-[var(--color-ink)] mb-4">Grunddaten</h2>

			<div class="space-y-4">
				<div>
					<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Name *</label>
					<Input bind:value={name} placeholder="z.B. Arbeitsvertrag Standard" required />
				</div>

				<div>
					<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Beschreibung</label>
					<textarea
						bind:value={description}
						class="input w-full min-h-[80px]"
						placeholder="Optionale Beschreibung der Vorlage..."
					></textarea>
				</div>
			</div>
		</Card>

		<!-- Signers -->
		<Card>
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-lg font-medium text-[var(--color-ink)]">Unterzeichner-Rollen</h2>
				<Button variant="secondary" size="sm" onclick={addSigner}>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="12" x2="12" y1="5" y2="19"/>
						<line x1="5" x2="19" y1="12" y2="12"/>
					</svg>
					Hinzufuegen
				</Button>
			</div>

			<p class="text-sm text-[var(--color-ink-muted)] mb-4">
				Definieren Sie die Rollen fuer Unterzeichner. Bei Verwendung der Vorlage werden die Platzhalter mit echten Daten gefuellt.
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
									<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Rolle *</label>
									<Input bind:value={signer.role} placeholder="z.B. Geschaeftsfuehrer" required />
								</div>
								<div>
									<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Standard-Name</label>
									<Input bind:value={signer.name} placeholder="Optional vorausfuellen" />
								</div>
								<div>
									<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Standard-E-Mail</label>
									<Input type="email" bind:value={signer.email} placeholder="Optional vorausfuellen" />
								</div>
								<div>
									<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Standard-Grund</label>
									<Input bind:value={signer.reason} placeholder="z.B. Genehmigung" />
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
			<h2 class="text-lg font-medium text-[var(--color-ink)] mb-4">Standardeinstellungen</h2>

			<div class="space-y-4">
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

				<label class="flex items-center gap-2">
					<input
						type="checkbox"
						bind:checked={autoRemind}
						class="rounded border-[var(--color-border)]"
					/>
					<span class="text-sm text-[var(--color-ink)]">Automatische Erinnerungen senden</span>
				</label>

				{#if autoRemind}
					<div class="ml-6">
						<label class="block text-sm font-medium text-[var(--color-ink)] mb-1">Erinnerung vor Ablauf (Tage)</label>
						<select bind:value={remindDays} class="input h-10 w-40">
							<option value={3}>3 Tage</option>
							<option value={5}>5 Tage</option>
							<option value={7}>7 Tage</option>
							<option value={14}>14 Tage</option>
						</select>
					</div>
				{/if}
			</div>
		</Card>

		<!-- Actions -->
		<div class="flex justify-end gap-3">
			<Button variant="secondary" href="/signatures/templates">Abbrechen</Button>
			<Button type="submit" disabled={!isValid || isSubmitting}>
				{#if isSubmitting}
					<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 12a9 9 0 1 1-6.219-8.56"/>
					</svg>
					Wird erstellt...
				{:else}
					Vorlage erstellen
				{/if}
			</Button>
		</div>
	</form>
</div>
