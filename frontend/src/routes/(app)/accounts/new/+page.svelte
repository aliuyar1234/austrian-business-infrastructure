<script lang="ts">
	import { goto } from '$app/navigation';
	import { accounts, type AccountType, type CreateAccountRequest } from '$lib/stores/accounts';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { z } from 'zod';

	// Account type selection
	let selectedType = $state<AccountType | null>(null);
	let step = $state(1);
	let isSubmitting = $state(false);

	// Form fields
	let name = $state('');
	let teilnehmerId = $state('');
	let benutzerId = $state('');
	let pin = $state('');
	let dienstgeberId = $state('');
	let eldaPassword = $state('');
	let fbUsername = $state('');
	let fbPassword = $state('');
	let tags = $state('');
	let errors = $state<Record<string, string>>({});

	const accountTypes = [
		{
			type: 'finanzonline' as AccountType,
			name: 'FinanzOnline',
			description: 'Access databox, submit UVA/ZM, retrieve documents',
			icon: '<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M7 7h10"/><path d="M7 12h10"/><path d="M7 17h10"/>',
		},
		{
			type: 'elda' as AccountType,
			name: 'ELDA',
			description: 'Employee registration, social security filings',
			icon: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>',
		},
		{
			type: 'firmenbuch' as AccountType,
			name: 'Firmenbuch',
			description: 'Company register search and extracts',
			icon: '<path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/><path d="M8 7h6"/><path d="M8 11h8"/>',
		},
	];

	// Schemas for each type
	const foSchema = z.object({
		name: z.string().min(1, 'Account name is required'),
		teilnehmerId: z.string().length(9, 'Teilnehmer-ID must be 9 digits'),
		benutzerId: z.string().min(1, 'Benutzer-ID is required'),
		pin: z.string().min(4, 'PIN must be at least 4 characters'),
	});

	const eldaSchema = z.object({
		name: z.string().min(1, 'Account name is required'),
		dienstgeberId: z.string().min(1, 'Dienstgeber-ID is required'),
		eldaPassword: z.string().min(1, 'Password is required'),
	});

	const fbSchema = z.object({
		name: z.string().min(1, 'Account name is required'),
		fbUsername: z.string().min(1, 'Username is required'),
		fbPassword: z.string().min(1, 'Password is required'),
	});

	function selectType(type: AccountType) {
		selectedType = type;
		step = 2;
	}

	function goBack() {
		if (step === 2) {
			step = 1;
			selectedType = null;
			errors = {};
		} else {
			goto('/accounts');
		}
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		errors = {};

		// Validate based on type
		let validation;
		if (selectedType === 'finanzonline') {
			validation = foSchema.safeParse({ name, teilnehmerId, benutzerId, pin });
		} else if (selectedType === 'elda') {
			validation = eldaSchema.safeParse({ name, dienstgeberId, eldaPassword });
		} else if (selectedType === 'firmenbuch') {
			validation = fbSchema.safeParse({ name, fbUsername, fbPassword });
		} else {
			return;
		}

		if (!validation.success) {
			const fieldErrors = validation.error.flatten().fieldErrors;
			errors = Object.fromEntries(
				Object.entries(fieldErrors).map(([k, v]) => [k, v?.[0] ?? ''])
			);
			return;
		}

		isSubmitting = true;

		// Build request
		const request: CreateAccountRequest = {
			name,
			type: selectedType,
			tags: tags ? tags.split(',').map((t) => t.trim()).filter(Boolean) : [],
		};

		if (selectedType === 'finanzonline') {
			request.teilnehmerId = teilnehmerId;
			request.benutzerId = benutzerId;
			request.pin = pin;
		} else if (selectedType === 'elda') {
			request.dienstgeberId = dienstgeberId;
			request.eldaPassword = eldaPassword;
		} else if (selectedType === 'firmenbuch') {
			request.username = fbUsername;
			request.password = fbPassword;
		}

		const result = await accounts.create(request);
		isSubmitting = false;

		if (result.success) {
			goto('/accounts');
		}
	}
</script>

<svelte:head>
	<title>Add Account - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-2xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<button onclick={goBack} aria-label="Go back" class="w-10 h-10 flex items-center justify-center rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors">
			<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m12 19-7-7 7-7M19 12H5"/>
			</svg>
		</button>
		<div>
			<h1 class="text-xl font-semibold text-[var(--color-ink)]">Add new account</h1>
			<p class="text-sm text-[var(--color-ink-muted)]">
				{#if step === 1}
					Select the type of account you want to connect
				{:else}
					Enter your {accountTypes.find(t => t.type === selectedType)?.name} credentials
				{/if}
			</p>
		</div>
	</div>

	{#if step === 1}
		<!-- Type selection -->
		<div class="grid gap-4">
			{#each accountTypes as type}
				<Card hover onclick={() => selectType(type.type)} class="group">
					<div class="flex items-center gap-4">
						<div class="w-12 h-12 rounded-xl bg-[var(--color-paper-inset)] group-hover:bg-[var(--color-accent-muted)] flex items-center justify-center transition-colors">
							<svg class="w-6 h-6 text-[var(--color-ink-muted)] group-hover:text-[var(--color-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
								{@html type.icon}
							</svg>
						</div>
						<div class="flex-1">
							<h3 class="font-medium text-[var(--color-ink)]">{type.name}</h3>
							<p class="text-sm text-[var(--color-ink-muted)]">{type.description}</p>
						</div>
						<svg class="w-5 h-5 text-[var(--color-ink-muted)] group-hover:text-[var(--color-accent)] transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="m9 18 6-6-6-6"/>
						</svg>
					</div>
				</Card>
			{/each}
		</div>
	{:else}
		<!-- Credential form -->
		<Card>
			<form onsubmit={handleSubmit} class="space-y-5">
				<!-- Account name -->
				<div>
					<label for="name" class="label">Account name</label>
					<Input
						type="text"
						id="name"
						bind:value={name}
						placeholder="e.g., Muster GmbH"
						error={errors.name}
					/>
					<p class="mt-1 text-xs text-[var(--color-ink-muted)]">A friendly name to identify this account</p>
				</div>

				{#if selectedType === 'finanzonline'}
					<div class="grid sm:grid-cols-2 gap-4">
						<div>
							<label for="teilnehmerId" class="label">Teilnehmer-ID</label>
							<Input
								type="text"
								id="teilnehmerId"
								bind:value={teilnehmerId}
								placeholder="123456789"
								maxlength={9}
								error={errors.teilnehmerId}
							/>
						</div>
						<div>
							<label for="benutzerId" class="label">Benutzer-ID</label>
							<Input
								type="text"
								id="benutzerId"
								bind:value={benutzerId}
								placeholder="Your user ID"
								error={errors.benutzerId}
							/>
						</div>
					</div>
					<div>
						<label for="pin" class="label">PIN / Passwort</label>
						<Input
							type="password"
							id="pin"
							bind:value={pin}
							placeholder="Enter your PIN"
							error={errors.pin}
						/>
						<p class="mt-1 text-xs text-[var(--color-ink-muted)]">Your credentials are encrypted and stored securely</p>
					</div>
				{:else if selectedType === 'elda'}
					<div>
						<label for="dienstgeberId" class="label">Dienstgeber-Kontonummer</label>
						<Input
							type="text"
							id="dienstgeberId"
							bind:value={dienstgeberId}
							placeholder="Your employer account number"
							error={errors.dienstgeberId}
						/>
					</div>
					<div>
						<label for="eldaPassword" class="label">Password</label>
						<Input
							type="password"
							id="eldaPassword"
							bind:value={eldaPassword}
							placeholder="Enter your ELDA password"
							error={errors.eldaPassword}
						/>
					</div>
				{:else if selectedType === 'firmenbuch'}
					<div>
						<label for="fbUsername" class="label">Username</label>
						<Input
							type="text"
							id="fbUsername"
							bind:value={fbUsername}
							placeholder="Your Firmenbuch username"
							error={errors.fbUsername}
						/>
					</div>
					<div>
						<label for="fbPassword" class="label">Password</label>
						<Input
							type="password"
							id="fbPassword"
							bind:value={fbPassword}
							placeholder="Enter your password"
							error={errors.fbPassword}
						/>
					</div>
				{/if}

				<!-- Tags -->
				<div>
					<label for="tags" class="label">Tags (optional)</label>
					<Input
						type="text"
						id="tags"
						bind:value={tags}
						placeholder="e.g., klient, 2024, steuerberater"
					/>
					<p class="mt-1 text-xs text-[var(--color-ink-muted)]">Comma-separated tags for organizing accounts</p>
				</div>

				<!-- Actions -->
				<div class="flex items-center justify-end gap-3 pt-2">
					<Button variant="secondary" type="button" onclick={goBack}>
						Cancel
					</Button>
					<Button type="submit" loading={isSubmitting}>
						Add account
					</Button>
				</div>
			</form>
		</Card>

		<!-- Security note -->
		<div class="flex gap-3 p-4 rounded-lg bg-[var(--color-info-muted)]">
			<svg class="w-5 h-5 text-[var(--color-info)] flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect width="18" height="11" x="3" y="11" rx="2" ry="2"/>
				<path d="M7 11V7a5 5 0 0 1 10 0v4"/>
			</svg>
			<div class="text-sm text-[var(--color-info)]">
				<p class="font-medium">Your credentials are secure</p>
				<p class="mt-0.5 opacity-80">All credentials are encrypted with AES-256-GCM before storage. We never store plaintext passwords.</p>
			</div>
		</div>
	{/if}
</div>
