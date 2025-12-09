<script lang="ts">
	import { page } from '$app/stores';
	import Button from '$lib/components/ui/Button.svelte';

	const errorMessages: Record<number, { title: string; description: string }> = {
		400: {
			title: 'Bad Request',
			description: 'The request could not be understood by the server.',
		},
		401: {
			title: 'Unauthorized',
			description: 'You need to be logged in to access this page.',
		},
		403: {
			title: 'Forbidden',
			description: "You don't have permission to access this resource.",
		},
		404: {
			title: 'Page Not Found',
			description: "The page you're looking for doesn't exist or has been moved.",
		},
		500: {
			title: 'Server Error',
			description: 'Something went wrong on our end. Please try again later.',
		},
		503: {
			title: 'Service Unavailable',
			description: "We're temporarily offline for maintenance. Please try again soon.",
		},
	};

	let statusCode = $derived($page.status);
	let errorInfo = $derived(
		errorMessages[statusCode] ?? {
			title: 'Something went wrong',
			description: 'An unexpected error occurred.',
		}
	);
</script>

<svelte:head>
	<title>{statusCode} - {errorInfo.title}</title>
</svelte:head>

<div class="min-h-screen bg-[var(--color-paper)] flex items-center justify-center p-6">
	<div class="max-w-md w-full text-center">
		<!-- Error icon -->
		<div class="w-20 h-20 mx-auto mb-6 rounded-full bg-[var(--color-error-muted)] flex items-center justify-center">
			{#if statusCode === 404}
				<svg class="w-10 h-10 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<circle cx="11" cy="11" r="8"/>
					<path d="m21 21-4.35-4.35"/>
					<path d="M11 8v4M11 16h.01"/>
				</svg>
			{:else if statusCode === 401 || statusCode === 403}
				<svg class="w-10 h-10 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<rect width="18" height="11" x="3" y="11" rx="2" ry="2"/>
					<path d="M7 11V7a5 5 0 0 1 10 0v4"/>
				</svg>
			{:else}
				<svg class="w-10 h-10 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
					<line x1="12" x2="12" y1="9" y2="13"/>
					<line x1="12" x2="12.01" y1="17" y2="17"/>
				</svg>
			{/if}
		</div>

		<!-- Status code -->
		<p class="text-6xl font-bold text-[var(--color-error)] mb-2">
			{statusCode}
		</p>

		<!-- Title -->
		<h1 class="text-xl font-semibold text-[var(--color-ink)] mb-2">
			{errorInfo.title}
		</h1>

		<!-- Description -->
		<p class="text-[var(--color-ink-muted)] mb-8">
			{errorInfo.description}
		</p>

		<!-- Actions -->
		<div class="flex flex-col sm:flex-row items-center justify-center gap-3">
			{#if statusCode === 401}
				<Button href="/login">
					Sign in
				</Button>
			{:else}
				<Button href="/">
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
						<polyline points="9 22 9 12 15 12 15 22"/>
					</svg>
					Go home
				</Button>
			{/if}
			<Button variant="secondary" onclick={() => window.history.back()}>
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="m12 19-7-7 7-7"/>
					<path d="M19 12H5"/>
				</svg>
				Go back
			</Button>
		</div>

		<!-- Debug info (dev only) -->
		{#if $page.error?.message && import.meta.env.DEV}
			<div class="mt-8 p-4 rounded-lg bg-[var(--color-paper-inset)] text-left">
				<p class="text-xs text-[var(--color-ink-muted)] uppercase tracking-wide mb-2">Debug Info</p>
				<pre class="text-xs text-[var(--color-error)] font-mono overflow-x-auto">{$page.error.message}</pre>
			</div>
		{/if}
	</div>
</div>
