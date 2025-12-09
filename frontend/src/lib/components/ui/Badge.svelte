<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props {
		variant?: 'default' | 'success' | 'warning' | 'error' | 'info';
		size?: 'sm' | 'md';
		dot?: boolean;
		class?: string;
		children: Snippet;
	}

	let {
		variant = 'default',
		size = 'md',
		dot = false,
		class: className,
		children
	}: Props = $props();

	const baseStyles = `
		inline-flex items-center gap-1.5
		font-medium rounded-full
		whitespace-nowrap
	`;

	const variants = {
		default: 'bg-[var(--color-paper-inset)] text-[var(--color-ink-secondary)]',
		success: 'bg-[var(--color-success-muted)] text-[var(--color-success)]',
		warning: 'bg-[var(--color-warning-muted)] text-[var(--color-warning)]',
		error: 'bg-[var(--color-error-muted)] text-[var(--color-error)]',
		info: 'bg-[var(--color-info-muted)] text-[var(--color-info)]'
	};

	const sizes = {
		sm: 'px-2 py-0.5 text-xs',
		md: 'px-2.5 py-1 text-xs'
	};

	const dotColors = {
		default: 'bg-[var(--color-ink-muted)]',
		success: 'bg-[var(--color-success)]',
		warning: 'bg-[var(--color-warning)]',
		error: 'bg-[var(--color-error)]',
		info: 'bg-[var(--color-info)]'
	};
</script>

<span class={cn(baseStyles, variants[variant], sizes[size], className)}>
	{#if dot}
		<span class={cn('w-1.5 h-1.5 rounded-full', dotColors[variant])}></span>
	{/if}
	{@render children()}
</span>
