<script lang="ts">
	import Card from './Card.svelte';
	import Skeleton from './Skeleton.svelte';

	interface Props {
		class?: string;
		variant?: 'list-item' | 'stat' | 'table-row' | 'document' | 'account';
	}

	let {
		class: className = '',
		variant = 'list-item',
	}: Props = $props();
</script>

{#if variant === 'list-item'}
	<Card class={className}>
		<div class="flex items-center gap-4">
			<Skeleton variant="circular" width="2.5rem" height="2.5rem" />
			<div class="flex-1 space-y-2">
				<Skeleton variant="text" width="60%" height="1rem" />
				<Skeleton variant="text" width="40%" height="0.75rem" />
			</div>
			<Skeleton variant="rectangular" width="4rem" height="1.5rem" />
		</div>
	</Card>
{:else if variant === 'stat'}
	<Card class={className}>
		<div class="space-y-3">
			<Skeleton variant="text" width="40%" height="0.75rem" />
			<Skeleton variant="text" width="60%" height="2rem" />
			<Skeleton variant="text" width="50%" height="0.625rem" />
		</div>
	</Card>
{:else if variant === 'table-row'}
	<div class="flex items-center gap-4 p-4 border-b border-black/4 {className}">
		<Skeleton variant="text" width="15%" height="1rem" />
		<Skeleton variant="text" width="25%" height="1rem" />
		<Skeleton variant="text" width="20%" height="1rem" />
		<Skeleton variant="text" width="15%" height="1rem" />
		<Skeleton variant="rectangular" width="4rem" height="1.5rem" />
	</div>
{:else if variant === 'document'}
	<Card class={className}>
		<div class="flex items-start gap-4">
			<Skeleton variant="rectangular" width="3rem" height="3rem" />
			<div class="flex-1 space-y-2">
				<Skeleton variant="text" width="70%" height="1rem" />
				<Skeleton variant="text" width="50%" height="0.75rem" />
				<Skeleton variant="text" width="30%" height="0.625rem" />
			</div>
		</div>
	</Card>
{:else if variant === 'account'}
	<Card class={className}>
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-4">
				<Skeleton variant="circular" width="3rem" height="3rem" />
				<div class="space-y-2">
					<Skeleton variant="text" width="10rem" height="1rem" />
					<Skeleton variant="text" width="8rem" height="0.75rem" />
				</div>
			</div>
			<div class="flex items-center gap-2">
				<Skeleton variant="rectangular" width="4rem" height="1.5rem" />
				<Skeleton variant="rectangular" width="2rem" height="2rem" />
			</div>
		</div>
	</Card>
{/if}
