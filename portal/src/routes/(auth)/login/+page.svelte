<script lang="ts">
	import { goto } from '$app/navigation';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { auth } from '$lib/stores/auth';

	let email = '';
	let password = '';
	let loading = false;
	let error = '';

	async function handleSubmit() {
		if (!email || !password) {
			error = 'Bitte alle Felder ausfüllen';
			return;
		}

		loading = true;
		error = '';

		try {
			const result = await auth.login(email, password);
			if (result.success) {
				goto('/');
			} else {
				error = result.error || 'Anmeldung fehlgeschlagen';
			}
		} catch (e: any) {
			error = e.message || 'Anmeldung fehlgeschlagen';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Anmelden | Mandantenportal</title>
</svelte:head>

<form on:submit|preventDefault={handleSubmit} class="space-y-6">
	<h2 class="text-2xl font-bold text-gray-900 text-center">Anmelden</h2>

	{#if error}
		<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
			{error}
		</div>
	{/if}

	<Input
		id="email"
		type="email"
		label="E-Mail"
		placeholder="ihre@email.at"
		bind:value={email}
		required
	/>

	<Input
		id="password"
		type="password"
		label="Passwort"
		placeholder="••••••••"
		bind:value={password}
		required
	/>

	<Button type="submit" class="w-full" {loading}>
		Anmelden
	</Button>
</form>
