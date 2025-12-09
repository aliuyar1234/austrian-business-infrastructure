<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { api } from '$lib/api/client';

	let token = '';
	let email = '';
	let name = '';
	let password = '';
	let confirmPassword = '';
	let loading = true;
	let activating = false;
	let error = '';
	let tokenValid = false;

	onMount(async () => {
		token = $page.params.token;

		try {
			const result = await api.validateToken(token);
			tokenValid = result.valid;
			email = result.email;
		} catch (e: any) {
			error = 'Dieser Aktivierungslink ist ungültig oder abgelaufen.';
		} finally {
			loading = false;
		}
	});

	async function handleSubmit() {
		if (password !== confirmPassword) {
			error = 'Passwörter stimmen nicht überein';
			return;
		}

		if (password.length < 8) {
			error = 'Passwort muss mindestens 8 Zeichen lang sein';
			return;
		}

		activating = true;
		error = '';

		try {
			await api.activate(token, password, name);
			goto('/login?activated=true');
		} catch (e: any) {
			error = e.message || 'Aktivierung fehlgeschlagen';
		} finally {
			activating = false;
		}
	}
</script>

<svelte:head>
	<title>Konto aktivieren | Mandantenportal</title>
</svelte:head>

{#if loading}
	<div class="flex justify-center py-8">
		<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
	</div>
{:else if !tokenValid}
	<div class="text-center py-8">
		<div class="text-red-500 text-5xl mb-4">✕</div>
		<h2 class="text-xl font-bold text-gray-900 mb-2">Link ungültig</h2>
		<p class="text-gray-600 mb-6">{error}</p>
		<a href="/login" class="text-primary hover:underline">Zur Anmeldung</a>
	</div>
{:else}
	<form on:submit|preventDefault={handleSubmit} class="space-y-6">
		<h2 class="text-2xl font-bold text-gray-900 text-center">Konto aktivieren</h2>

		<p class="text-sm text-gray-600 text-center">
			Willkommen! Bitte vervollständigen Sie Ihr Profil.
		</p>

		{#if error}
			<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
				{error}
			</div>
		{/if}

		<Input
			id="email"
			type="email"
			label="E-Mail"
			value={email}
			disabled
		/>

		<Input
			id="name"
			type="text"
			label="Ihr Name"
			placeholder="Max Mustermann"
			bind:value={name}
			required
		/>

		<Input
			id="password"
			type="password"
			label="Passwort"
			placeholder="Mindestens 8 Zeichen"
			bind:value={password}
			required
		/>

		<Input
			id="confirmPassword"
			type="password"
			label="Passwort bestätigen"
			placeholder="Passwort wiederholen"
			bind:value={confirmPassword}
			required
		/>

		<Button type="submit" class="w-full" loading={activating}>
			Konto aktivieren
		</Button>
	</form>
{/if}
