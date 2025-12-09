<script lang="ts">
	import { Settings, User, Bell, Shield } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { auth } from '$lib/stores/auth';
	import { branding } from '$lib/stores/branding';

	let name = $auth.client?.name || '';
	let email = $auth.client?.email || '';
	let currentPassword = '';
	let newPassword = '';
	let confirmPassword = '';
	let saving = false;
	let passwordError = '';
	let successMessage = '';

	async function saveProfile() {
		saving = true;
		successMessage = '';
		// TODO: Implement profile update API
		await new Promise((r) => setTimeout(r, 1000));
		successMessage = 'Profil gespeichert';
		saving = false;
	}

	async function changePassword() {
		passwordError = '';

		if (newPassword !== confirmPassword) {
			passwordError = 'Passwörter stimmen nicht überein';
			return;
		}

		if (newPassword.length < 8) {
			passwordError = 'Passwort muss mindestens 8 Zeichen lang sein';
			return;
		}

		saving = true;
		// TODO: Implement password change API
		await new Promise((r) => setTimeout(r, 1000));
		successMessage = 'Passwort geändert';
		currentPassword = '';
		newPassword = '';
		confirmPassword = '';
		saving = false;
	}
</script>

<svelte:head>
	<title>Einstellungen | {$branding.company_name}</title>
</svelte:head>

<div class="max-w-2xl mx-auto space-y-6">
	<h1 class="text-2xl font-bold text-gray-900">Einstellungen</h1>

	{#if successMessage}
		<div class="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg">
			{successMessage}
		</div>
	{/if}

	<!-- Profile -->
	<Card>
		<div class="flex items-center gap-3 mb-6">
			<div class="p-2 bg-primary/10 rounded-lg">
				<User class="w-5 h-5 text-primary" />
			</div>
			<h2 class="text-lg font-semibold text-gray-900">Profil</h2>
		</div>

		<form on:submit|preventDefault={saveProfile} class="space-y-4">
			<Input
				id="name"
				label="Name"
				bind:value={name}
			/>

			<Input
				id="email"
				type="email"
				label="E-Mail"
				bind:value={email}
				disabled
			/>
			<p class="text-xs text-gray-500 -mt-2">
				E-Mail-Änderungen bitte beim Support anfragen.
			</p>

			<div class="pt-2">
				<Button type="submit" loading={saving}>
					Speichern
				</Button>
			</div>
		</form>
	</Card>

	<!-- Password -->
	<Card>
		<div class="flex items-center gap-3 mb-6">
			<div class="p-2 bg-primary/10 rounded-lg">
				<Shield class="w-5 h-5 text-primary" />
			</div>
			<h2 class="text-lg font-semibold text-gray-900">Passwort ändern</h2>
		</div>

		<form on:submit|preventDefault={changePassword} class="space-y-4">
			{#if passwordError}
				<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
					{passwordError}
				</div>
			{/if}

			<Input
				id="currentPassword"
				type="password"
				label="Aktuelles Passwort"
				bind:value={currentPassword}
			/>

			<Input
				id="newPassword"
				type="password"
				label="Neues Passwort"
				placeholder="Mindestens 8 Zeichen"
				bind:value={newPassword}
			/>

			<Input
				id="confirmPassword"
				type="password"
				label="Passwort bestätigen"
				bind:value={confirmPassword}
			/>

			<div class="pt-2">
				<Button type="submit" loading={saving}>
					Passwort ändern
				</Button>
			</div>
		</form>
	</Card>

	<!-- Notifications (placeholder) -->
	<Card>
		<div class="flex items-center gap-3 mb-6">
			<div class="p-2 bg-primary/10 rounded-lg">
				<Bell class="w-5 h-5 text-primary" />
			</div>
			<h2 class="text-lg font-semibold text-gray-900">Benachrichtigungen</h2>
		</div>

		<div class="space-y-4">
			<label class="flex items-center justify-between">
				<div>
					<p class="font-medium text-gray-900">E-Mail bei neuen Dokumenten</p>
					<p class="text-sm text-gray-500">Benachrichtigung wenn neue Dokumente geteilt werden</p>
				</div>
				<input type="checkbox" checked class="w-5 h-5 text-primary rounded" />
			</label>

			<label class="flex items-center justify-between">
				<div>
					<p class="font-medium text-gray-900">E-Mail bei Freigabeanfragen</p>
					<p class="text-sm text-gray-500">Benachrichtigung bei neuen Freigabeanforderungen</p>
				</div>
				<input type="checkbox" checked class="w-5 h-5 text-primary rounded" />
			</label>

			<label class="flex items-center justify-between">
				<div>
					<p class="font-medium text-gray-900">E-Mail bei neuen Nachrichten</p>
					<p class="text-sm text-gray-500">Benachrichtigung bei neuen Chat-Nachrichten</p>
				</div>
				<input type="checkbox" checked class="w-5 h-5 text-primary rounded" />
			</label>
		</div>
	</Card>

	<!-- Support -->
	{#if $branding.support_email || $branding.support_phone}
		<Card>
			<h2 class="text-lg font-semibold text-gray-900 mb-4">Support</h2>
			<div class="space-y-2 text-gray-600">
				{#if $branding.support_email}
					<p>
						E-Mail: <a href="mailto:{$branding.support_email}" class="text-primary hover:underline">
							{$branding.support_email}
						</a>
					</p>
				{/if}
				{#if $branding.support_phone}
					<p>
						Telefon: <a href="tel:{$branding.support_phone}" class="text-primary hover:underline">
							{$branding.support_phone}
						</a>
					</p>
				{/if}
			</div>
		</Card>
	{/if}
</div>
