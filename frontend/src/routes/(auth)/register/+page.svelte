<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth, authError, isLoading } from '$lib/stores/auth';
	import { config } from '$lib/config';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { z } from 'zod';

	// Password policy: min 8 chars, 1 uppercase, 1 lowercase, 1 number
	const passwordPolicy = z
		.string()
		.min(8, 'Password must be at least 8 characters')
		.regex(/[A-Z]/, 'Password must contain at least one uppercase letter')
		.regex(/[a-z]/, 'Password must contain at least one lowercase letter')
		.regex(/[0-9]/, 'Password must contain at least one number');

	const registerSchema = z.object({
		name: z.string().min(2, 'Name must be at least 2 characters'),
		email: z.string().email('Please enter a valid email address'),
		company: z.string().min(1, 'Company name is required'),
		password: passwordPolicy,
		confirmPassword: z.string()
	}).refine((data) => data.password === data.confirmPassword, {
		message: 'Passwords do not match',
		path: ['confirmPassword']
	});

	let name = $state('');
	let email = $state('');
	let company = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let acceptTerms = $state(false);
	let errors = $state<Record<string, string>>({});

	// Password strength indicator
	let passwordStrength = $derived(() => {
		if (!password) return { score: 0, label: '', color: '' };
		let score = 0;
		if (password.length >= 8) score++;
		if (password.length >= 12) score++;
		if (/[A-Z]/.test(password)) score++;
		if (/[a-z]/.test(password)) score++;
		if (/[0-9]/.test(password)) score++;
		if (/[^A-Za-z0-9]/.test(password)) score++;

		if (score <= 2) return { score: 1, label: 'Weak', color: 'var(--color-error)' };
		if (score <= 4) return { score: 2, label: 'Fair', color: 'var(--color-warning)' };
		return { score: 3, label: 'Strong', color: 'var(--color-success)' };
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		errors = {};
		auth.clearError();

		if (!acceptTerms) {
			errors = { terms: 'You must accept the terms and conditions' };
			return;
		}

		const result = registerSchema.safeParse({ name, email, company, password, confirmPassword });
		if (!result.success) {
			const fieldErrors = result.error.flatten().fieldErrors;
			errors = {
				name: fieldErrors.name?.[0],
				email: fieldErrors.email?.[0],
				company: fieldErrors.company?.[0],
				password: fieldErrors.password?.[0],
				confirmPassword: fieldErrors.confirmPassword?.[0]
			};
			return;
		}

		const response = await auth.register({
			name,
			email,
			password,
			tenantName: company
		});

		if (response.success) {
			goto('/');
		}
	}

	function handleOAuth(provider: 'google' | 'microsoft') {
		window.location.href = `${config.api.baseUrl}/api/v1/auth/oauth/${provider}?signup=true`;
	}
</script>

<svelte:head>
	<title>Create account - Austrian Business Infrastructure</title>
</svelte:head>

<div class="space-y-6">
	<!-- Mobile logo -->
	<div class="lg:hidden flex items-center justify-center gap-3">
		<div class="w-10 h-10 rounded-lg bg-[var(--color-accent)] flex items-center justify-center">
			<span class="text-white font-bold text-lg">ABI</span>
		</div>
	</div>

	<!-- Header -->
	<div class="text-center lg:text-left">
		<h2 class="text-2xl font-bold text-[var(--color-ink)]">Create your account</h2>
		<p class="mt-2 text-[var(--color-ink-muted)]">
			Start managing your Austrian business filings
		</p>
	</div>

	<!-- OAuth buttons -->
	{#if config.oauth.google.enabled || config.oauth.microsoft.enabled}
		<div class="space-y-3">
			{#if config.oauth.google.enabled}
				<button
					onclick={() => handleOAuth('google')}
					class="
						w-full h-11 flex items-center justify-center gap-3
						bg-white border border-black/10 rounded-lg
						text-sm font-medium text-[var(--color-ink)]
						hover:bg-[var(--color-paper-inset)] hover:border-black/15
						transition-all duration-150
					"
				>
					<svg class="w-5 h-5" viewBox="0 0 24 24">
						<path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
						<path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
						<path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
						<path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
					</svg>
					Sign up with Google
				</button>
			{/if}

			{#if config.oauth.microsoft.enabled}
				<button
					onclick={() => handleOAuth('microsoft')}
					class="
						w-full h-11 flex items-center justify-center gap-3
						bg-white border border-black/10 rounded-lg
						text-sm font-medium text-[var(--color-ink)]
						hover:bg-[var(--color-paper-inset)] hover:border-black/15
						transition-all duration-150
					"
				>
					<svg class="w-5 h-5" viewBox="0 0 23 23">
						<path fill="#f35325" d="M1 1h10v10H1z"/>
						<path fill="#81bc06" d="M12 1h10v10H12z"/>
						<path fill="#05a6f0" d="M1 12h10v10H1z"/>
						<path fill="#ffba08" d="M12 12h10v10H12z"/>
					</svg>
					Sign up with Microsoft
				</button>
			{/if}

			<div class="relative">
				<div class="absolute inset-0 flex items-center">
					<div class="w-full border-t border-black/10"></div>
				</div>
				<div class="relative flex justify-center text-xs">
					<span class="px-3 bg-[var(--color-paper)] text-[var(--color-ink-muted)]">
						or continue with email
					</span>
				</div>
			</div>
		</div>
	{/if}

	<!-- Registration form -->
	<form onsubmit={handleSubmit} class="space-y-4">
		<div>
			<label for="name" class="label">Full name</label>
			<Input
				type="text"
				id="name"
				bind:value={name}
				placeholder="Johann Mustermann"
				autocomplete="name"
				error={errors.name}
			/>
		</div>

		<div>
			<label for="email" class="label">Work email</label>
			<Input
				type="email"
				id="email"
				bind:value={email}
				placeholder="johann@unternehmen.at"
				autocomplete="email"
				error={errors.email}
			/>
		</div>

		<div>
			<label for="company" class="label">Company name</label>
			<Input
				type="text"
				id="company"
				bind:value={company}
				placeholder="Mustermann GmbH"
				autocomplete="organization"
				error={errors.company}
			/>
		</div>

		<div>
			<label for="password" class="label">Password</label>
			<Input
				type="password"
				id="password"
				bind:value={password}
				placeholder="Create a strong password"
				autocomplete="new-password"
				error={errors.password}
			/>
			{#if password}
				<div class="mt-2 flex items-center gap-2">
					<div class="flex-1 h-1.5 bg-black/5 rounded-full overflow-hidden">
						<div
							class="h-full transition-all duration-300"
							style="width: {passwordStrength().score * 33.33}%; background: {passwordStrength().color}"
						></div>
					</div>
					<span class="text-xs font-medium" style="color: {passwordStrength().color}">
						{passwordStrength().label}
					</span>
				</div>
				<p class="mt-1.5 text-xs text-[var(--color-ink-muted)]">
					Use at least 8 characters with uppercase, lowercase, and numbers
				</p>
			{/if}
		</div>

		<div>
			<label for="confirmPassword" class="label">Confirm password</label>
			<Input
				type="password"
				id="confirmPassword"
				bind:value={confirmPassword}
				placeholder="Confirm your password"
				autocomplete="new-password"
				error={errors.confirmPassword}
			/>
		</div>

		<!-- Terms acceptance -->
		<div class="flex items-start gap-3">
			<input
				type="checkbox"
				id="terms"
				bind:checked={acceptTerms}
				class="mt-0.5 w-4 h-4 rounded border-black/20 text-[var(--color-accent)] focus:ring-[var(--color-accent)]"
			/>
			<label for="terms" class="text-sm text-[var(--color-ink-muted)]">
				I agree to the
				<a href="/terms" class="text-[var(--color-accent)] hover:underline">Terms of Service</a>
				and
				<a href="/privacy" class="text-[var(--color-accent)] hover:underline">Privacy Policy</a>
			</label>
		</div>
		{#if errors.terms}
			<p class="text-xs text-[var(--color-error)]">{errors.terms}</p>
		{/if}

		<!-- Error message -->
		{#if $authError}
			<div class="p-3 rounded-lg bg-[var(--color-error-muted)] text-sm text-[var(--color-error)]">
				{$authError}
			</div>
		{/if}

		<Button type="submit" loading={$isLoading} class="w-full">
			Create account
		</Button>
	</form>

	<!-- Login link -->
	<p class="text-center text-sm text-[var(--color-ink-muted)]">
		Already have an account?
		<a href="/login" class="font-medium text-[var(--color-accent)] hover:underline">
			Sign in
		</a>
	</p>
</div>
