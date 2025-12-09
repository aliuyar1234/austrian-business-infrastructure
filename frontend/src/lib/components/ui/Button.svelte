<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props {
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'link';
		size?: 'sm' | 'md' | 'lg';
		disabled?: boolean;
		loading?: boolean;
		type?: 'button' | 'submit' | 'reset';
		href?: string;
		class?: string;
		children: Snippet;
		onclick?: (e: MouseEvent) => void;
	}

	let {
		variant = 'primary',
		size = 'md',
		disabled = false,
		loading = false,
		type = 'button',
		href,
		class: className,
		children,
		onclick
	}: Props = $props();

	const baseStyles = `
		inline-flex items-center justify-center gap-2
		font-medium transition-all duration-150
		focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2
		disabled:opacity-50 disabled:pointer-events-none
	`;

	const variants = {
		primary: `
			bg-[var(--color-accent)] text-white
			hover:bg-[var(--color-accent-hover)]
			focus-visible:ring-[var(--color-accent)]
			active:scale-[0.98]
		`,
		secondary: `
			bg-[var(--color-paper-elevated)] text-[var(--color-ink)]
			border border-black/10
			hover:bg-[var(--color-paper-inset)] hover:border-black/15
			focus-visible:ring-[var(--color-ink)]
		`,
		ghost: `
			bg-transparent text-[var(--color-ink-secondary)]
			hover:bg-[var(--color-paper-inset)] hover:text-[var(--color-ink)]
			focus-visible:ring-[var(--color-ink)]
		`,
		danger: `
			bg-[var(--color-error)] text-white
			hover:bg-red-700
			focus-visible:ring-[var(--color-error)]
		`,
		link: `
			bg-transparent text-[var(--color-accent)]
			hover:underline underline-offset-4
			focus-visible:ring-[var(--color-accent)]
			p-0 h-auto
		`
	};

	const sizes = {
		sm: 'h-8 px-3 text-sm rounded-md',
		md: 'h-10 px-4 text-sm rounded-md',
		lg: 'h-12 px-6 text-base rounded-lg'
	};
</script>

{#if href}
	<a
		{href}
		class={cn(baseStyles, variants[variant], sizes[size], className)}
		class:pointer-events-none={disabled || loading}
	>
		{#if loading}
			<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
				<circle
					class="opacity-25"
					cx="12"
					cy="12"
					r="10"
					stroke="currentColor"
					stroke-width="4"
				></circle>
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
				></path>
			</svg>
		{/if}
		{@render children()}
	</a>
{:else}
	<button
		{type}
		disabled={disabled || loading}
		class={cn(baseStyles, variants[variant], sizes[size], className)}
		onclick={onclick}
	>
		{#if loading}
			<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
				<circle
					class="opacity-25"
					cx="12"
					cy="12"
					r="10"
					stroke="currentColor"
					stroke-width="4"
				></circle>
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
				></path>
			</svg>
		{/if}
		{@render children()}
	</button>
{/if}
