<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props {
		padding?: 'none' | 'sm' | 'md' | 'lg';
		hover?: boolean;
		class?: string;
		children: Snippet;
		onclick?: () => void;
	}

	let {
		padding = 'md',
		hover = false,
		class: className,
		children,
		onclick
	}: Props = $props();

	const baseStyles = `
		bg-[var(--color-paper-elevated)]
		border border-black/6
		rounded-lg
		shadow-[var(--shadow-card)]
	`;

	const paddingStyles = {
		none: '',
		sm: 'p-4',
		md: 'p-6',
		lg: 'p-8'
	};

	const hoverStyles = `
		transition-all duration-200
		hover:shadow-[var(--shadow-md)]
		hover:border-black/10
		cursor-pointer
	`;
</script>

{#if onclick}
	<button
		class={cn(baseStyles, paddingStyles[padding], hover && hoverStyles, 'w-full text-left', className)}
		onclick={onclick}
	>
		{@render children()}
	</button>
{:else}
	<div class={cn(baseStyles, paddingStyles[padding], hover && hoverStyles, className)}>
		{@render children()}
	</div>
{/if}
