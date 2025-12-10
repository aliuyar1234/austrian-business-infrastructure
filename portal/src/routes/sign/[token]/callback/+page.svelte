<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	type CallbackState = 'processing' | 'success' | 'error';

	let state = $state<CallbackState>('processing');
	let errorMessage = $state('');

	onMount(async () => {
		const token = $page.params.token;
		const searchParams = $page.url.searchParams;
		const code = searchParams.get('code');
		const authState = searchParams.get('state');
		const error = searchParams.get('error');

		if (error) {
			state = 'error';
			errorMessage = getErrorMessage(error);
			return;
		}

		if (!code) {
			state = 'error';
			errorMessage = 'Authentifizierungscode fehlt. Bitte versuchen Sie es erneut.';
			return;
		}

		try {
			// TODO: Call actual API to complete the signing
			// const response = await fetch(`/api/v1/sign/${token}/callback?code=${code}&state=${authState}`, {
			//   method: 'POST'
			// });
			// const result = await response.json();

			// Simulate processing
			await new Promise(resolve => setTimeout(resolve, 2000));

			// Redirect to success state on the signing page
			state = 'success';
			setTimeout(() => {
				goto(`/sign/${token}?completed=true`);
			}, 1500);
		} catch (e: any) {
			state = 'error';
			errorMessage = e.message || 'Fehler bei der Verarbeitung der Signatur';
		}
	});

	function getErrorMessage(error: string): string {
		switch (error) {
			case 'access_denied':
				return 'Die Authentifizierung wurde abgebrochen.';
			case 'login_required':
				return 'Sie muessen sich bei ID Austria anmelden.';
			case 'consent_required':
				return 'Die Zustimmung zur Datenfreigabe ist erforderlich.';
			case 'interaction_required':
				return 'Eine Interaktion ist erforderlich. Bitte versuchen Sie es erneut.';
			case 'invalid_request':
				return 'Ungueltige Anfrage. Bitte starten Sie den Vorgang neu.';
			default:
				return 'Ein Fehler ist bei der Authentifizierung aufgetreten.';
		}
	}

	function retry() {
		const token = $page.params.token;
		goto(`/sign/${token}`);
	}
</script>

<svelte:head>
	<title>Signatur wird verarbeitet | ID Austria</title>
</svelte:head>

<div class="min-h-screen bg-gray-50 flex flex-col">
	<!-- Header -->
	<header class="bg-white border-b border-gray-200">
		<div class="max-w-3xl mx-auto px-4 py-4">
			<div class="flex items-center gap-3">
				<div class="w-10 h-10 bg-blue-600 rounded-lg flex items-center justify-center">
					<svg class="w-6 h-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
					</svg>
				</div>
				<div>
					<h1 class="font-semibold text-gray-900">Digitale Signatur</h1>
					<p class="text-sm text-gray-500">Qualifizierte elektronische Signatur mit ID Austria</p>
				</div>
			</div>
		</div>
	</header>

	<!-- Main content -->
	<main class="flex-1 flex items-center justify-center py-8">
		<div class="max-w-md mx-auto px-4 w-full">
			{#if state === 'processing'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="relative">
						<div class="animate-spin rounded-full h-16 w-16 border-4 border-blue-200 border-t-blue-600 mx-auto"></div>
						<div class="absolute inset-0 flex items-center justify-center">
							<svg class="w-6 h-6 text-blue-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
							</svg>
						</div>
					</div>
					<h2 class="mt-6 text-xl font-semibold text-gray-900">Signatur wird erstellt</h2>
					<p class="mt-2 text-gray-600">
						Bitte warten Sie, waehrend Ihre qualifizierte elektronische Signatur erstellt wird...
					</p>
					<div class="mt-6 space-y-2 text-sm text-gray-500">
						<div class="flex items-center justify-center gap-2">
							<svg class="w-4 h-4 text-green-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<polyline points="20 6 9 17 4 12"/>
							</svg>
							ID Austria Authentifizierung erfolgreich
						</div>
						<div class="flex items-center justify-center gap-2">
							<div class="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
							Signatur wird erstellt...
						</div>
					</div>
				</div>

			{:else if state === 'success'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto">
						<svg class="w-8 h-8 text-green-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="20 6 9 17 4 12"/>
						</svg>
					</div>
					<h2 class="mt-6 text-xl font-semibold text-gray-900">Signatur erfolgreich!</h2>
					<p class="mt-2 text-gray-600">
						Das Dokument wurde erfolgreich signiert.
					</p>
					<p class="mt-4 text-sm text-gray-500">
						Sie werden weitergeleitet...
					</p>
				</div>

			{:else if state === 'error'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto">
						<svg class="w-8 h-8 text-red-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="12" cy="12" r="10"/>
							<line x1="15" x2="9" y1="9" y2="15"/>
							<line x1="9" x2="15" y1="9" y2="15"/>
						</svg>
					</div>
					<h2 class="mt-6 text-xl font-semibold text-gray-900">Fehler</h2>
					<p class="mt-2 text-gray-600">{errorMessage}</p>
					<button
						class="mt-6 px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
						onclick={retry}
					>
						Erneut versuchen
					</button>
				</div>
			{/if}
		</div>
	</main>

	<!-- Footer -->
	<footer class="bg-white border-t border-gray-200 py-4">
		<div class="max-w-3xl mx-auto px-4 text-center text-sm text-gray-500">
			<div class="flex items-center justify-center gap-2">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<rect width="18" height="11" x="3" y="11" rx="2" ry="2"/>
					<path d="M7 11V7a5 5 0 0 1 10 0v4"/>
				</svg>
				Sichere Verbindung
			</div>
		</div>
	</footer>
</div>
