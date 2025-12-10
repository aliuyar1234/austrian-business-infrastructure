<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import Button from '$lib/components/ui/Button.svelte';

	type SigningState = 'loading' | 'valid' | 'invalid' | 'expired' | 'already_signed' | 'signing' | 'success' | 'error';

	interface SigningInfo {
		documentTitle: string;
		documentName: string;
		requesterName: string;
		requesterCompany: string;
		signerName: string;
		signerEmail: string;
		reason?: string;
		message?: string;
		expiresAt: Date;
		signersCount: number;
		currentSignerIndex: number;
	}

	let token = '';
	let state = $state<SigningState>('loading');
	let info = $state<SigningInfo | null>(null);
	let errorMessage = $state('');
	let showPreview = $state(false);

	onMount(async () => {
		token = $page.params.token;
		await loadSigningInfo();
	});

	async function loadSigningInfo() {
		try {
			// TODO: Call actual API
			// const response = await fetch(`/api/v1/sign/${token}`);
			// const data = await response.json();

			// Mock data
			await new Promise(resolve => setTimeout(resolve, 800));

			// Simulate different states based on token
			if (token === 'expired') {
				state = 'expired';
				return;
			}
			if (token === 'invalid') {
				state = 'invalid';
				return;
			}
			if (token === 'signed') {
				state = 'already_signed';
				return;
			}

			info = {
				documentTitle: 'Arbeitsvertrag Max Mustermann',
				documentName: 'arbeitsvertrag_mustermann.pdf',
				requesterName: 'Maria Beispiel',
				requesterCompany: 'Beispiel Steuerberatung GmbH',
				signerName: 'Max Mustermann',
				signerEmail: 'max@example.com',
				reason: 'Zustimmung',
				message: 'Bitte unterschreiben Sie den beigefuegten Arbeitsvertrag bis zum 31.12.2024.',
				expiresAt: new Date(Date.now() + 14 * 86400000),
				signersCount: 2,
				currentSignerIndex: 1
			};
			state = 'valid';
		} catch (e: any) {
			state = 'error';
			errorMessage = e.message || 'Ein Fehler ist aufgetreten';
		}
	}

	async function startSigning() {
		state = 'signing';

		try {
			// TODO: Redirect to ID Austria authentication
			// The backend will create the OIDC authorization URL
			// const response = await fetch(`/api/v1/sign/${token}/auth`);
			// const { authUrl } = await response.json();
			// window.location.href = authUrl;

			// Mock: Simulate redirect
			await new Promise(resolve => setTimeout(resolve, 1500));

			// In reality, user would be redirected to ID Austria
			// After successful auth, they'd be redirected back to a callback URL
			// For demo, simulate success
			state = 'success';
		} catch (e: any) {
			state = 'error';
			errorMessage = e.message || 'Fehler beim Starten der Signatur';
		}
	}

	function formatDate(date: Date): string {
		return new Intl.DateTimeFormat('de-AT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		}).format(date);
	}

	let daysLeft = $derived(info ? Math.ceil((info.expiresAt.getTime() - Date.now()) / 86400000) : 0);
</script>

<svelte:head>
	<title>Dokument signieren | ID Austria</title>
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
	<main class="flex-1 py-8">
		<div class="max-w-3xl mx-auto px-4">
			{#if state === 'loading'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
					<p class="mt-4 text-gray-600">Signaturanfrage wird geladen...</p>
				</div>

			{:else if state === 'invalid'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto">
						<svg class="w-8 h-8 text-red-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="12" cy="12" r="10"/>
							<line x1="15" x2="9" y1="9" y2="15"/>
							<line x1="9" x2="15" y1="9" y2="15"/>
						</svg>
					</div>
					<h2 class="mt-4 text-xl font-semibold text-gray-900">Link ungueltig</h2>
					<p class="mt-2 text-gray-600">
						Dieser Signatur-Link ist ungueltig oder wurde bereits verwendet.
					</p>
					<p class="mt-4 text-sm text-gray-500">
						Bitte kontaktieren Sie den Absender fuer einen neuen Link.
					</p>
				</div>

			{:else if state === 'expired'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="w-16 h-16 bg-orange-100 rounded-full flex items-center justify-center mx-auto">
						<svg class="w-8 h-8 text-orange-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="12" cy="12" r="10"/>
							<polyline points="12 6 12 12 16 14"/>
						</svg>
					</div>
					<h2 class="mt-4 text-xl font-semibold text-gray-900">Link abgelaufen</h2>
					<p class="mt-2 text-gray-600">
						Die Frist fuer diese Signaturanfrage ist leider abgelaufen.
					</p>
					<p class="mt-4 text-sm text-gray-500">
						Bitte kontaktieren Sie den Absender fuer eine neue Anfrage.
					</p>
				</div>

			{:else if state === 'already_signed'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto">
						<svg class="w-8 h-8 text-green-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="20 6 9 17 4 12"/>
						</svg>
					</div>
					<h2 class="mt-4 text-xl font-semibold text-gray-900">Bereits signiert</h2>
					<p class="mt-2 text-gray-600">
						Sie haben dieses Dokument bereits signiert.
					</p>
				</div>

			{:else if state === 'success'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mx-auto">
						<svg class="w-10 h-10 text-green-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M9 12l2 2 4-4"/>
							<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
						</svg>
					</div>
					<h2 class="mt-6 text-2xl font-semibold text-gray-900">Signatur erfolgreich!</h2>
					<p class="mt-2 text-gray-600">
						Das Dokument wurde erfolgreich mit Ihrer qualifizierten elektronischen Signatur versehen.
					</p>
					{#if info && info.currentSignerIndex < info.signersCount}
						<p class="mt-4 text-sm text-gray-500">
							Das Dokument wird nun an den naechsten Unterzeichner weitergeleitet.
						</p>
					{:else}
						<p class="mt-4 text-sm text-gray-500">
							Der Absender wird ueber den Abschluss informiert.
						</p>
					{/if}
					<div class="mt-8 pt-6 border-t border-gray-200">
						<p class="text-sm text-gray-500">Sie koennen dieses Fenster jetzt schliessen.</p>
					</div>
				</div>

			{:else if state === 'error'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto">
						<svg class="w-8 h-8 text-red-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="12" cy="12" r="10"/>
							<line x1="12" x2="12" y1="8" y2="12"/>
							<line x1="12" x2="12.01" y1="16" y2="16"/>
						</svg>
					</div>
					<h2 class="mt-4 text-xl font-semibold text-gray-900">Fehler</h2>
					<p class="mt-2 text-gray-600">{errorMessage}</p>
					<Button class="mt-6" onclick={loadSigningInfo}>
						Erneut versuchen
					</Button>
				</div>

			{:else if state === 'signing'}
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
					<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
					<h2 class="mt-6 text-xl font-semibold text-gray-900">Weiterleitung zu ID Austria</h2>
					<p class="mt-2 text-gray-600">
						Sie werden zur sicheren Authentifizierung weitergeleitet...
					</p>
					<div class="mt-6 flex items-center justify-center gap-2 text-sm text-gray-500">
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<rect width="18" height="11" x="3" y="11" rx="2" ry="2"/>
							<path d="M7 11V7a5 5 0 0 1 10 0v4"/>
						</svg>
						Sichere Verbindung
					</div>
				</div>

			{:else if state === 'valid' && info}
				<!-- Document info -->
				<div class="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
					<!-- Document header -->
					<div class="px-6 py-4 bg-gray-50 border-b border-gray-200">
						<div class="flex items-start gap-4">
							<div class="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center flex-shrink-0">
								<svg class="w-6 h-6 text-blue-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
									<polyline points="14 2 14 8 20 8"/>
								</svg>
							</div>
							<div class="flex-1 min-w-0">
								<h2 class="text-lg font-semibold text-gray-900">{info.documentTitle}</h2>
								<p class="text-sm text-gray-500">{info.documentName}</p>
							</div>
							<button
								class="text-blue-600 hover:text-blue-700 text-sm font-medium"
								onclick={() => showPreview = !showPreview}
							>
								{showPreview ? 'Vorschau schliessen' : 'Dokument ansehen'}
							</button>
						</div>
					</div>

					{#if showPreview}
						<div class="border-b border-gray-200 bg-gray-100 p-4">
							<div class="aspect-[3/4] bg-white rounded shadow-inner flex items-center justify-center">
								<p class="text-gray-400">Dokumentvorschau wird geladen...</p>
							</div>
						</div>
					{/if}

					<!-- Request details -->
					<div class="p-6">
						<div class="flex items-start gap-3 mb-6">
							<div class="w-10 h-10 bg-gray-100 rounded-full flex items-center justify-center flex-shrink-0">
								<svg class="w-5 h-5 text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
									<circle cx="12" cy="7" r="4"/>
								</svg>
							</div>
							<div>
								<p class="text-sm text-gray-500">Anfrage von</p>
								<p class="font-medium text-gray-900">{info.requesterName}</p>
								<p class="text-sm text-gray-500">{info.requesterCompany}</p>
							</div>
						</div>

						{#if info.message}
							<div class="mb-6 p-4 bg-blue-50 rounded-lg">
								<p class="text-sm font-medium text-blue-900 mb-1">Nachricht:</p>
								<p class="text-sm text-blue-800">{info.message}</p>
							</div>
						{/if}

						<div class="grid gap-4 sm:grid-cols-2">
							<div>
								<p class="text-sm text-gray-500">Ihr Name</p>
								<p class="font-medium text-gray-900">{info.signerName}</p>
							</div>
							<div>
								<p class="text-sm text-gray-500">E-Mail</p>
								<p class="font-medium text-gray-900">{info.signerEmail}</p>
							</div>
							{#if info.reason}
								<div>
									<p class="text-sm text-gray-500">Signaturgrund</p>
									<p class="font-medium text-gray-900">{info.reason}</p>
								</div>
							{/if}
							<div>
								<p class="text-sm text-gray-500">Gueltig bis</p>
								<p class="font-medium {daysLeft <= 3 ? 'text-red-600' : daysLeft <= 7 ? 'text-orange-600' : 'text-gray-900'}">
									{formatDate(info.expiresAt)}
									<span class="text-sm font-normal">({daysLeft} Tage)</span>
								</p>
							</div>
						</div>

						{#if info.signersCount > 1}
							<div class="mt-6 pt-4 border-t border-gray-200">
								<p class="text-sm text-gray-500">
									Unterzeichner {info.currentSignerIndex} von {info.signersCount}
								</p>
								<div class="mt-2 flex gap-1">
									{#each Array(info.signersCount) as _, i}
										<div class="flex-1 h-2 rounded-full {i < info.currentSignerIndex ? 'bg-green-500' : i === info.currentSignerIndex - 1 ? 'bg-blue-500' : 'bg-gray-200'}"></div>
									{/each}
								</div>
							</div>
						{/if}
					</div>

					<!-- Actions -->
					<div class="px-6 py-4 bg-gray-50 border-t border-gray-200">
						<div class="flex flex-col sm:flex-row gap-3">
							<Button class="flex-1 justify-center" onclick={startSigning}>
								<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
								</svg>
								Mit ID Austria signieren
							</Button>
						</div>
						<p class="mt-3 text-xs text-center text-gray-500">
							Sie werden zur Authentifizierung mit ID Austria weitergeleitet.
							<br/>
							Die Signatur entspricht den eIDAS-Anforderungen fuer qualifizierte elektronische Signaturen (QES).
						</p>
					</div>
				</div>

				<!-- Info box -->
				<div class="mt-6 bg-blue-50 rounded-lg p-4">
					<div class="flex gap-3">
						<svg class="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="12" cy="12" r="10"/>
							<path d="M12 16v-4"/>
							<path d="M12 8h.01"/>
						</svg>
						<div class="text-sm text-blue-800">
							<p class="font-medium">Was ist eine qualifizierte elektronische Signatur?</p>
							<p class="mt-1">
								Eine qualifizierte elektronische Signatur (QES) ist rechtlich einer handschriftlichen
								Unterschrift gleichgestellt und erfuellt die hoechsten Sicherheitsanforderungen der EU-Verordnung eIDAS.
							</p>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</main>

	<!-- Footer -->
	<footer class="bg-white border-t border-gray-200 py-4">
		<div class="max-w-3xl mx-auto px-4 text-center text-sm text-gray-500">
			<p>Qualifizierte elektronische Signatur gemaess eIDAS-Verordnung (EU) Nr. 910/2014</p>
			<p class="mt-1">Powered by A-Trust und ID Austria</p>
		</div>
	</footer>
</div>
