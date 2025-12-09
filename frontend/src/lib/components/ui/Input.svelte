<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		type?: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url' | 'search';
		value?: string;
		placeholder?: string;
		disabled?: boolean;
		readonly?: boolean;
		required?: boolean;
		name?: string;
		id?: string;
		autocomplete?: string;
		error?: string;
		class?: string;
		oninput?: (e: Event & { currentTarget: HTMLInputElement }) => void;
		onchange?: (e: Event & { currentTarget: HTMLInputElement }) => void;
		onblur?: (e: FocusEvent & { currentTarget: HTMLInputElement }) => void;
	}

	let {
		type = 'text',
		value = $bindable(''),
		placeholder,
		disabled = false,
		readonly = false,
		required = false,
		name,
		id,
		autocomplete,
		error,
		class: className,
		oninput,
		onchange,
		onblur
	}: Props = $props();

	const baseStyles = `
		w-full h-10 px-3
		bg-[var(--color-paper-elevated)]
		border border-black/10 rounded-md
		text-[var(--color-ink)] text-sm
		placeholder:text-[var(--color-ink-muted)]
		transition-all duration-150
		focus:outline-none focus:border-[var(--color-accent)]
		focus:ring-3 focus:ring-[var(--color-accent-muted)]
		disabled:opacity-50 disabled:cursor-not-allowed
		read-only:bg-[var(--color-paper-inset)]
	`;

	const errorStyles = `
		border-[var(--color-error)]
		focus:border-[var(--color-error)]
		focus:ring-[var(--color-error-muted)]
	`;
</script>

<div class="relative">
	<input
		{type}
		bind:value
		{placeholder}
		{disabled}
		{readonly}
		{required}
		{name}
		{id}
		{autocomplete}
		class={cn(baseStyles, error && errorStyles, className)}
		oninput={oninput}
		onchange={onchange}
		onblur={onblur}
	/>
	{#if error}
		<p class="mt-1.5 text-xs text-[var(--color-error)]">{error}</p>
	{/if}
</div>
