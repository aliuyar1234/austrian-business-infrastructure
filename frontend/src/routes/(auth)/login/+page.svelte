<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth, authError, isLoading } from '$lib/stores/auth';
	import { config } from '$lib/config';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { z } from 'zod';

	// Form schema
	const loginSchema = z.object({
		email: z.string().email('Please enter a valid email address'),
		password: z.string().min(1, 'Password is required')
	});

	let email = $state('');
	let password = $state('');
	let errors = $state<{ email?: string; password?: string }>({});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();

		// Clear previous errors
		errors = {};
		auth.clearError();

		// Validate
		const result = loginSchema.safeParse({ email, password });
		if (!result.success) {
			const fieldErrors = result.error.flatten().fieldErrors;
			errors = {
				email: fieldErrors.email?.[0],
				password: fieldErrors.password?.[0]
			};
			return;
		}

		// Attempt login
		const response = await auth.login(email, password);
		if (response.success) {
			goto('/');
		}
	}

	function handleOAuth(provider: 'google' | 'microsoft') {
		// Redirect to OAuth provider
		window.location.href = `${config.api.baseUrl}/api/v1/auth/oauth/${provider}`;
	}
</script>

<svelte:head>
	<title>Sign in - Austrian Business Infrastructure</title>
</svelte:head>

<div class="space-y-8">
	<!-- Mobile logo -->
	<div class="lg:hidden flex items-center justify-center gap-3">
		<div class="w-10 h-10 rounded-lg bg-[var(--color-accent)] flex items-center justify-center">
			<span class="text-white font-bold text-lg">ABI</span>
		</div>
	</div>

	<!-- Header -->
	<div class="text-center lg:text-left">
		<h2 class="text-2xl font-bold text-[var(--color-ink)]">Welcome back</h2>
		<p class="mt-2 text-[var(--color-ink-muted)]">
			Sign in to your account to continue
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
					Continue with Google
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
					Continue with Microsoft
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

	<!-- Login form -->
	<form onsubmit={handleSubmit} class="space-y-5">
		<div class="space-y-4">
			<div>
				<label for="email" class="label">Email address</label>
				<Input
					type="email"
					id="email"
					bind:value={email}
					placeholder="you@company.at"
					autocomplete="email"
					error={errors.email}
				/>
			</div>

			<div>
				<div class="flex items-center justify-between mb-1.5">
					<label for="password" class="label !mb-0">Password</label>
					<a
						href="/forgot-password"
						class="text-xs font-medium text-[var(--color-accent)] hover:underline"
					>
						Forgot password?
					</a>
				</div>
				<Input
					type="password"
					id="password"
					bind:value={password}
					placeholder="Enter your password"
					autocomplete="current-password"
					error={errors.password}
				/>
			</div>
		</div>

		<!-- Error message -->
		{#if $authError}
			<div class="p-3 rounded-lg bg-[var(--color-error-muted)] text-sm text-[var(--color-error)]">
				{$authError}
			</div>
		{/if}

		<Button type="submit" loading={$isLoading} class="w-full">
			Sign in
		</Button>
	</form>

	<!-- Register link -->
	<p class="text-center text-sm text-[var(--color-ink-muted)]">
		Don't have an account?
		<a href="/register" class="font-medium text-[var(--color-accent)] hover:underline">
			Create one
		</a>
	</p>
</div>
