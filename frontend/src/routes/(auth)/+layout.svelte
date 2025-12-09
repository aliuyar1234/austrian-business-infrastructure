<script lang="ts">
	import { goto } from '$app/navigation';
	import { isAuthenticated } from '$lib/stores/auth';

	let { children } = $props();

	// Redirect to dashboard if already authenticated
	$effect(() => {
		if ($isAuthenticated) {
			goto('/');
		}
	});
</script>

{#if !$isAuthenticated}
	<div class="min-h-screen bg-[var(--color-paper)] flex">
		<!-- Left side - Branding -->
		<div class="hidden lg:flex lg:w-1/2 xl:w-2/5 bg-[var(--color-navy)] relative overflow-hidden">
			<!-- Geometric pattern background -->
			<div class="absolute inset-0 opacity-10">
				<svg class="w-full h-full" viewBox="0 0 100 100" preserveAspectRatio="none">
					<defs>
						<pattern id="grid" width="10" height="10" patternUnits="userSpaceOnUse">
							<path d="M 10 0 L 0 0 0 10" fill="none" stroke="white" stroke-width="0.5"/>
						</pattern>
					</defs>
					<rect width="100" height="100" fill="url(#grid)" />
				</svg>
			</div>

			<!-- Austrian red accent stripe -->
			<div class="absolute left-0 top-0 bottom-0 w-2 bg-[var(--color-accent)]"></div>

			<!-- Content -->
			<div class="relative z-10 flex flex-col justify-between p-12 text-white">
				<!-- Logo -->
				<div class="flex items-center gap-3">
					<div class="w-10 h-10 rounded-lg bg-[var(--color-accent)] flex items-center justify-center">
						<span class="font-bold text-lg">ABI</span>
					</div>
					<div>
						<p class="font-semibold">Austrian Business</p>
						<p class="text-sm text-white/70">Infrastructure</p>
					</div>
				</div>

				<!-- Main message -->
				<div class="space-y-6">
					<h1 class="text-4xl xl:text-5xl font-bold leading-tight text-balance">
						All Austrian government APIs.
						<span class="text-[var(--color-accent)]">One platform.</span>
					</h1>
					<p class="text-lg text-white/70 max-w-md">
						FinanzOnline, ELDA, Firmenbuch, and more. Streamline your business operations with unified access to Austrian government services.
					</p>
				</div>

				<!-- Features list -->
				<div class="space-y-4">
					<div class="flex items-center gap-3 text-sm text-white/80">
						<svg class="w-5 h-5 text-[var(--color-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="20 6 9 17 4 12"/>
						</svg>
						<span>Automated tax filing (UVA, ZM)</span>
					</div>
					<div class="flex items-center gap-3 text-sm text-white/80">
						<svg class="w-5 h-5 text-[var(--color-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="20 6 9 17 4 12"/>
						</svg>
						<span>EN16931-compliant electronic invoicing</span>
					</div>
					<div class="flex items-center gap-3 text-sm text-white/80">
						<svg class="w-5 h-5 text-[var(--color-accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<polyline points="20 6 9 17 4 12"/>
						</svg>
						<span>Real-time document synchronization</span>
					</div>
				</div>
			</div>
		</div>

		<!-- Right side - Form -->
		<div class="flex-1 flex items-center justify-center p-8">
			<div class="w-full max-w-md">
				{@render children()}
			</div>
		</div>
	</div>
{/if}
