<script lang="ts">
	import { clsx } from 'clsx';

	export let type: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url' = 'text';
	export let value = '';
	export let placeholder = '';
	export let disabled = false;
	export let error = '';
	export let label = '';
	export let id = '';
	export let required = false;
</script>

<div class="w-full">
	{#if label}
		<label for={id} class="block text-sm font-medium text-gray-700 mb-1">
			{label}
			{#if required}
				<span class="text-red-500">*</span>
			{/if}
		</label>
	{/if}

	<input
		{id}
		{type}
		{placeholder}
		{disabled}
		{required}
		bind:value
		class={clsx(
			'block w-full rounded-lg border px-3 py-2 text-sm',
			'focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary',
			'disabled:bg-gray-50 disabled:text-gray-500 disabled:cursor-not-allowed',
			error ? 'border-red-300 focus:border-red-500 focus:ring-red-500/50' : 'border-gray-300',
			$$props.class
		)}
		on:input
		on:change
		on:blur
		on:focus
	/>

	{#if error}
		<p class="mt-1 text-sm text-red-600">{error}</p>
	{/if}
</div>
