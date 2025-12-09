<script lang="ts">
	interface Props {
		class?: string;
		variant?: 'text' | 'circular' | 'rectangular';
		width?: string;
		height?: string;
		lines?: number;
	}

	let {
		class: className = '',
		variant = 'rectangular',
		width,
		height,
		lines = 1,
	}: Props = $props();

	const baseClasses = 'animate-pulse bg-[var(--color-paper-inset)]';

	const variantClasses = {
		text: 'rounded',
		circular: 'rounded-full',
		rectangular: 'rounded-lg',
	};

	const defaultSizes = {
		text: { width: '100%', height: '1rem' },
		circular: { width: '2.5rem', height: '2.5rem' },
		rectangular: { width: '100%', height: '4rem' },
	};

	const computedWidth = width ?? defaultSizes[variant].width;
	const computedHeight = height ?? defaultSizes[variant].height;
</script>

{#if lines > 1}
	<div class="space-y-2 {className}">
		{#each Array(lines) as _, i}
			<div
				class="{baseClasses} {variantClasses[variant]}"
				style="width: {i === lines - 1 ? '75%' : computedWidth}; height: {computedHeight}"
			></div>
		{/each}
	</div>
{:else}
	<div
		class="{baseClasses} {variantClasses[variant]} {className}"
		style="width: {computedWidth}; height: {computedHeight}"
	></div>
{/if}
