<script lang="ts">
	import { clsx } from 'clsx';

	export let variant: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let disabled = false;
	export let loading = false;
	export let type: 'button' | 'submit' | 'reset' = 'button';

	const variants = {
		primary: 'bg-primary text-white hover:bg-primary/90',
		secondary: 'bg-gray-100 text-gray-900 hover:bg-gray-200',
		outline: 'border border-gray-300 bg-white text-gray-700 hover:bg-gray-50',
		ghost: 'text-gray-700 hover:bg-gray-100',
		danger: 'bg-red-600 text-white hover:bg-red-700'
	};

	const sizes = {
		sm: 'px-3 py-1.5 text-sm',
		md: 'px-4 py-2 text-sm',
		lg: 'px-6 py-3 text-base'
	};
</script>

<button
	{type}
	disabled={disabled || loading}
	class={clsx(
		'inline-flex items-center justify-center rounded-lg font-medium transition-colors',
		'focus:outline-none focus:ring-2 focus:ring-primary/50 focus:ring-offset-2',
		'disabled:opacity-50 disabled:cursor-not-allowed',
		variants[variant],
		sizes[size],
		$$props.class
	)}
	on:click
>
	{#if loading}
		<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
			<path
				class="opacity-75"
				fill="currentColor"
				d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
			/>
		</svg>
	{/if}
	<slot />
</button>
