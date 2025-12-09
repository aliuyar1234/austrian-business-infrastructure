<script lang="ts">
	import { clsx } from 'clsx';

	export let padding: 'none' | 'sm' | 'md' | 'lg' = 'md';

	const paddings = {
		none: '',
		sm: 'p-3',
		md: 'p-4',
		lg: 'p-6'
	};
</script>

<div
	class={clsx(
		'bg-white rounded-lg border border-gray-200 shadow-sm',
		paddings[padding],
		$$props.class
	)}
>
	<slot />
</div>
