<script lang="ts">
	import { formatDate } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	type VerificationStatus = 'valid' | 'invalid' | 'indeterminate';

	interface SignatureInfo {
		signerName: string;
		signerEmail?: string;
		signedAt: Date;
		isValid: boolean;
		reason?: string;
		certificate?: {
			subject: string;
			issuer: string;
			validFrom: Date;
			validTo: Date;
			isQualified: boolean;
		};
	}

	interface VerificationResult {
		isValid: boolean;
		status: VerificationStatus;
		documentHash: string;
		signatureCount: number;
		signatures: SignatureInfo[];
		warnings: string[];
		errors: string[];
		verifiedAt: Date;
	}

	let selectedFile = $state<File | null>(null);
	let isVerifying = $state(false);
	let result = $state<VerificationResult | null>(null);
	let error = $state<string | null>(null);

	function handleFileSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files && input.files[0]) {
			selectedFile = input.files[0];
			result = null;
			error = null;
		}
	}

	function handleFileDrop(event: DragEvent) {
		event.preventDefault();
		if (event.dataTransfer?.files && event.dataTransfer.files[0]) {
			const file = event.dataTransfer.files[0];
			if (file.type === 'application/pdf') {
				selectedFile = file;
				result = null;
				error = null;
			}
		}
	}

	async function verifyDocument() {
		if (!selectedFile || isVerifying) return;

		isVerifying = true;
		error = null;

		try {
			// TODO: Call actual API
			// const formData = new FormData();
			// formData.append('file', selectedFile);
			// const response = await fetch('/api/v1/verify', { method: 'POST', body: formData });
			// result = await response.json();

			// Mock result
			await new Promise(resolve => setTimeout(resolve, 1500));

			result = {
				isValid: true,
				status: 'valid',
				documentHash: 'a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0',
				signatureCount: 2,
				signatures: [
					{
						signerName: 'Max Mustermann',
						signerEmail: 'max@example.com',
						signedAt: new Date(Date.now() - 172800000),
						isValid: true,
						reason: 'Zustimmung',
						certificate: {
							subject: 'CN=Max Mustermann, O=Test GmbH, C=AT',
							issuer: 'CN=A-Trust Qualified CA, O=A-Trust, C=AT',
							validFrom: new Date(Date.now() - 365 * 86400000),
							validTo: new Date(Date.now() + 365 * 86400000),
							isQualified: true
						}
					},
					{
						signerName: 'Anna Schmidt',
						signerEmail: 'anna@example.com',
						signedAt: new Date(Date.now() - 86400000),
						isValid: true,
						reason: 'Genehmigung',
						certificate: {
							subject: 'CN=Anna Schmidt, O=Test GmbH, C=AT',
							issuer: 'CN=A-Trust Qualified CA, O=A-Trust, C=AT',
							validFrom: new Date(Date.now() - 180 * 86400000),
							validTo: new Date(Date.now() + 545 * 86400000),
							isQualified: true
						}
					}
				],
				warnings: [],
				errors: [],
				verifiedAt: new Date()
			};
		} catch (err) {
			error = 'Fehler bei der Verifizierung. Bitte versuchen Sie es erneut.';
		} finally {
			isVerifying = false;
		}
	}

	function getStatusLabel(status: VerificationStatus): string {
		switch (status) {
			case 'valid': return 'Gueltig';
			case 'invalid': return 'Ungueltig';
			case 'indeterminate': return 'Unbestimmt';
		}
	}

	function getStatusVariant(status: VerificationStatus): 'success' | 'error' | 'warning' {
		switch (status) {
			case 'valid': return 'success';
			case 'invalid': return 'error';
			case 'indeterminate': return 'warning';
		}
	}

	function reset() {
		selectedFile = null;
		result = null;
		error = null;
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
			<h1 class="text-2xl font-semibold text-[var(--color-ink)]">Signatur verifizieren</h1>
			<p class="text-[var(--color-ink-muted)]">Pruefen Sie die Gueltigkeit von digitalen Signaturen</p>
		</div>
	</div>

	{#if !result}
		<!-- Upload area -->
		<Card>
			{#if selectedFile}
				<div class="flex items-center gap-4 p-4 bg-[var(--color-paper-inset)] rounded-lg mb-4">
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
					<Button variant="ghost" size="sm" onclick={reset}>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<line x1="18" x2="6" y1="6" y2="18"/>
							<line x1="6" x2="18" y1="6" y2="18"/>
						</svg>
					</Button>
				</div>

				<div class="flex justify-center">
					<Button onclick={verifyDocument} disabled={isVerifying}>
						{#if isVerifying}
							<svg class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M21 12a9 9 0 1 1-6.219-8.56"/>
							</svg>
							Wird geprueft...
						{:else}
							<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M9 12l2 2 4-4"/>
								<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
							</svg>
							Signatur pruefen
						{/if}
					</Button>
				</div>
			{:else}
				<div
					role="button"
					tabindex="0"
					class="border-2 border-dashed border-[var(--color-border)] rounded-lg p-12 text-center hover:border-[var(--color-accent)] transition-colors cursor-pointer"
					ondrop={handleFileDrop}
					ondragover={(e) => e.preventDefault()}
					onclick={() => document.getElementById('file-input')?.click()}
					onkeydown={(e) => e.key === 'Enter' && document.getElementById('file-input')?.click()}
				>
					<svg class="w-16 h-16 mx-auto text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
						<path d="M9 12l2 2 4-4"/>
						<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
					</svg>
					<p class="mt-4 text-lg text-[var(--color-ink)]">Signiertes PDF-Dokument hochladen</p>
					<p class="mt-2 text-sm text-[var(--color-ink-muted)]">
						Ziehen Sie ein Dokument hierher oder klicken Sie zum Auswaehlen
					</p>
				</div>
				<input
					id="file-input"
					type="file"
					accept=".pdf,application/pdf"
					class="hidden"
					onchange={handleFileSelect}
				/>
			{/if}

			{#if error}
				<div class="mt-4 p-4 bg-[var(--color-error-muted)] rounded-lg">
					<p class="text-sm text-[var(--color-error)]">{error}</p>
				</div>
			{/if}
		</Card>
	{:else}
		<!-- Verification result -->
		<Card class="text-center py-8">
			<div class="w-20 h-20 mx-auto rounded-full flex items-center justify-center {
				result.status === 'valid' ? 'bg-[var(--color-success-muted)]' :
				result.status === 'invalid' ? 'bg-[var(--color-error-muted)]' :
				'bg-[var(--color-warning-muted)]'
			}">
				{#if result.status === 'valid'}
					<svg class="w-10 h-10 text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M9 12l2 2 4-4"/>
						<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
					</svg>
				{:else if result.status === 'invalid'}
					<svg class="w-10 h-10 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
						<line x1="15" x2="9" y1="9" y2="15"/>
						<line x1="9" x2="15" y1="9" y2="15"/>
					</svg>
				{:else}
					<svg class="w-10 h-10 text-[var(--color-warning)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
						<line x1="12" x2="12" y1="8" y2="12"/>
						<line x1="12" x2="12.01" y1="16" y2="16"/>
					</svg>
				{/if}
			</div>

			<h2 class="mt-4 text-2xl font-semibold text-[var(--color-ink)]">
				{result.status === 'valid' ? 'Signatur gueltig' :
				 result.status === 'invalid' ? 'Signatur ungueltig' :
				 'Verifizierung unvollstaendig'}
			</h2>

			<p class="mt-2 text-[var(--color-ink-muted)]">
				{result.signatureCount} Signatur{result.signatureCount !== 1 ? 'en' : ''} gefunden
			</p>

			<div class="mt-4 flex justify-center gap-4">
				<Badge variant={getStatusVariant(result.status)} size="lg">
					{getStatusLabel(result.status)}
				</Badge>
			</div>
		</Card>

		<!-- Signatures detail -->
		{#if result.signatures.length > 0}
			<Card>
				<h3 class="text-lg font-medium text-[var(--color-ink)] mb-4">Signaturen im Dokument</h3>

				<div class="space-y-4">
					{#each result.signatures as sig, index}
						<div class="p-4 bg-[var(--color-paper-inset)] rounded-lg">
							<div class="flex items-start justify-between gap-4">
								<div class="flex items-center gap-3">
									<div class="w-10 h-10 rounded-full flex items-center justify-center {
										sig.isValid ? 'bg-[var(--color-success-muted)]' : 'bg-[var(--color-error-muted)]'
									}">
										{#if sig.isValid}
											<svg class="w-5 h-5 text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
												<polyline points="20 6 9 17 4 12"/>
											</svg>
										{:else}
											<svg class="w-5 h-5 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
												<line x1="18" x2="6" y1="6" y2="18"/>
												<line x1="6" x2="18" y1="6" y2="18"/>
											</svg>
										{/if}
									</div>
									<div>
										<div class="font-medium text-[var(--color-ink)]">{sig.signerName}</div>
										{#if sig.signerEmail}
											<div class="text-sm text-[var(--color-ink-muted)]">{sig.signerEmail}</div>
										{/if}
									</div>
								</div>
								<Badge variant={sig.isValid ? 'success' : 'error'} size="sm">
									{sig.isValid ? 'Gueltig' : 'Ungueltig'}
								</Badge>
							</div>

							<div class="mt-4 grid gap-3 sm:grid-cols-2 text-sm">
								<div>
									<span class="text-[var(--color-ink-muted)]">Signiert am:</span>
									<span class="ml-2 text-[var(--color-ink)]">{formatDate(sig.signedAt)}</span>
								</div>
								{#if sig.reason}
									<div>
										<span class="text-[var(--color-ink-muted)]">Grund:</span>
										<span class="ml-2 text-[var(--color-ink)]">{sig.reason}</span>
									</div>
								{/if}
							</div>

							{#if sig.certificate}
								<details class="mt-4">
									<summary class="text-sm text-[var(--color-accent)] cursor-pointer hover:underline">
										Zertifikatsdetails anzeigen
									</summary>
									<div class="mt-3 p-3 bg-[var(--color-paper)] rounded text-sm space-y-2">
										<div>
											<span class="text-[var(--color-ink-muted)]">Inhaber:</span>
											<span class="ml-2 text-[var(--color-ink)] font-mono text-xs">{sig.certificate.subject}</span>
										</div>
										<div>
											<span class="text-[var(--color-ink-muted)]">Aussteller:</span>
											<span class="ml-2 text-[var(--color-ink)] font-mono text-xs">{sig.certificate.issuer}</span>
										</div>
										<div>
											<span class="text-[var(--color-ink-muted)]">Gueltig:</span>
											<span class="ml-2 text-[var(--color-ink)]">
												{formatDate(sig.certificate.validFrom)} - {formatDate(sig.certificate.validTo)}
											</span>
										</div>
										<div class="flex items-center gap-2">
											<span class="text-[var(--color-ink-muted)]">Qualifiziert (QES):</span>
											{#if sig.certificate.isQualified}
												<Badge variant="success" size="sm">Ja</Badge>
											{:else}
												<Badge variant="default" size="sm">Nein</Badge>
											{/if}
										</div>
									</div>
								</details>
							{/if}
						</div>
					{/each}
				</div>
			</Card>
		{/if}

		<!-- Warnings and errors -->
		{#if result.warnings.length > 0 || result.errors.length > 0}
			<Card>
				<h3 class="text-lg font-medium text-[var(--color-ink)] mb-4">Hinweise</h3>

				{#if result.errors.length > 0}
					<div class="space-y-2 mb-4">
						{#each result.errors as err}
							<div class="flex items-start gap-2 text-sm">
								<svg class="w-4 h-4 text-[var(--color-error)] flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<circle cx="12" cy="12" r="10"/>
									<line x1="15" x2="9" y1="9" y2="15"/>
									<line x1="9" x2="15" y1="9" y2="15"/>
								</svg>
								<span class="text-[var(--color-error)]">{err}</span>
							</div>
						{/each}
					</div>
				{/if}

				{#if result.warnings.length > 0}
					<div class="space-y-2">
						{#each result.warnings as warning}
							<div class="flex items-start gap-2 text-sm">
								<svg class="w-4 h-4 text-[var(--color-warning)] flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
									<line x1="12" x2="12" y1="9" y2="13"/>
									<line x1="12" x2="12.01" y1="17" y2="17"/>
								</svg>
								<span class="text-[var(--color-warning)]">{warning}</span>
							</div>
						{/each}
					</div>
				{/if}
			</Card>
		{/if}

		<!-- Document info -->
		<Card>
			<h3 class="text-lg font-medium text-[var(--color-ink)] mb-4">Dokumentinformationen</h3>

			<dl class="grid gap-4 sm:grid-cols-2 text-sm">
				<div>
					<dt class="text-[var(--color-ink-muted)]">Dateiname</dt>
					<dd class="font-medium text-[var(--color-ink)]">{selectedFile?.name}</dd>
				</div>
				<div>
					<dt class="text-[var(--color-ink-muted)]">Geprueft am</dt>
					<dd class="font-medium text-[var(--color-ink)]">{formatDate(result.verifiedAt)}</dd>
				</div>
				<div class="sm:col-span-2">
					<dt class="text-[var(--color-ink-muted)]">Dokument-Hash (SHA-256)</dt>
					<dd class="font-mono text-xs text-[var(--color-ink)] break-all">{result.documentHash}</dd>
				</div>
			</dl>
		</Card>

		<!-- Actions -->
		<div class="flex justify-center gap-3">
			<Button variant="secondary" onclick={reset}>
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
					<path d="M3 3v5h5"/>
				</svg>
				Neues Dokument pruefen
			</Button>
		</div>
	{/if}
</div>
