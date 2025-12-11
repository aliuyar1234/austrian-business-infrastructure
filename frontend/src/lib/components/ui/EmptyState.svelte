<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';
	import Card from './Card.svelte';
	import Button from './Button.svelte';

	interface Props {
		/** Title text displayed below the icon */
		title: string;
		/** Description text displayed below the title */
		description?: string;
		/** URL for the action button */
		actionHref?: string;
		/** Label for the action button */
		actionLabel?: string;
		/** Callback for the action button (alternative to href) */
		onAction?: () => void;
		/** Additional CSS classes */
		class?: string;
		/** Icon snippet to display at the top */
		icon?: Snippet;
	}

	let {
		title,
		description,
		actionHref,
		actionLabel,
		onAction,
		class: className,
		icon
	}: Props = $props();
</script>

<Card class={cn('text-center py-12', className)}>
	{#if icon}
		<div class="w-12 h-12 mx-auto text-[var(--color-ink-muted)]">
			{@render icon()}
		</div>
	{/if}
	<h3 class="mt-4 text-lg font-medium text-[var(--color-ink)]">{title}</h3>
	{#if description}
		<p class="mt-2 text-[var(--color-ink-muted)]">{description}</p>
	{/if}
	{#if actionLabel && (actionHref || onAction)}
		{#if actionHref}
			<Button href={actionHref} class="mt-6">{actionLabel}</Button>
		{:else if onAction}
			<Button onclick={onAction} class="mt-6">{actionLabel}</Button>
		{/if}
	{/if}
</Card>
