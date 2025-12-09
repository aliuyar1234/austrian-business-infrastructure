<script lang="ts">
	import { auth, authError, isLoading } from '$lib/stores/auth';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { z } from 'zod';

	const emailSchema = z.object({
		email: z.string().email('Please enter a valid email address')
	});

	let email = $state('');
	let errors = $state<{ email?: string }>({});
	let submitted = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		errors = {};
		auth.clearError();

		const result = emailSchema.safeParse({ email });
		if (!result.success) {
			errors = { email: result.error.flatten().fieldErrors.email?.[0] };
			return;
		}

		const response = await auth.requestPasswordReset(email);
		if (response.success) {
			submitted = true;
		}
	}
</script>

<svelte:head>
	<title>Reset password - Austrian Business Infrastructure</title>
</svelte:head>

<div class="space-y-8">
	<!-- Mobile logo -->
	<div class="lg:hidden flex items-center justify-center gap-3">
		<div class="w-10 h-10 rounded-lg bg-[var(--color-accent)] flex items-center justify-center">
			<span class="text-white font-bold text-lg">ABI</span>
		</div>
	</div>

	{#if submitted}
		<!-- Success state -->
		<div class="text-center lg:text-left">
			<div class="w-16 h-16 mx-auto lg:mx-0 rounded-full bg-[var(--color-success-muted)] flex items-center justify-center mb-6">
				<svg class="w-8 h-8 text-[var(--color-success)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M22 10.5V6a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2v12c0 1.1.9 2 2 2h12.5"/>
					<path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/>
					<path d="m16 19 2 2 4-4"/>
				</svg>
			</div>
			<h2 class="text-2xl font-bold text-[var(--color-ink)]">Check your email</h2>
			<p class="mt-3 text-[var(--color-ink-muted)]">
				We've sent a password reset link to<br/>
				<span class="font-medium text-[var(--color-ink)]">{email}</span>
			</p>
			<p class="mt-4 text-sm text-[var(--color-ink-muted)]">
				Didn't receive the email? Check your spam folder or
				<button
					onclick={() => { submitted = false; }}
					class="text-[var(--color-accent)] hover:underline"
				>
					try again
				</button>
			</p>
		</div>

		<Button variant="secondary" href="/login" class="w-full">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m12 19-7-7 7-7"/>
				<path d="M19 12H5"/>
			</svg>
			Back to sign in
		</Button>
	{:else}
		<!-- Form state -->
		<div class="text-center lg:text-left">
			<h2 class="text-2xl font-bold text-[var(--color-ink)]">Reset your password</h2>
			<p class="mt-2 text-[var(--color-ink-muted)]">
				Enter your email address and we'll send you a link to reset your password
			</p>
		</div>

		<form onsubmit={handleSubmit} class="space-y-5">
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

			{#if $authError}
				<div class="p-3 rounded-lg bg-[var(--color-error-muted)] text-sm text-[var(--color-error)]">
					{$authError}
				</div>
			{/if}

			<Button type="submit" loading={$isLoading} class="w-full">
				Send reset link
			</Button>
		</form>

		<p class="text-center text-sm text-[var(--color-ink-muted)]">
			Remember your password?
			<a href="/login" class="font-medium text-[var(--color-accent)] hover:underline">
				Sign in
			</a>
		</p>
	{/if}
</div>
