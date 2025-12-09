<script lang="ts">
	import { clsx } from 'clsx';

	export let variant: 'default' | 'success' | 'warning' | 'danger' | 'info' = 'default';

	const variants = {
		default: 'bg-gray-100 text-gray-800',
		success: 'bg-green-100 text-green-800',
		warning: 'bg-yellow-100 text-yellow-800',
		danger: 'bg-red-100 text-red-800',
		info: 'bg-blue-100 text-blue-800'
	};
</script>

<span
	class={clsx(
		'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
		variants[variant],
		$$props.class
	)}
>
	<slot />
</span>
