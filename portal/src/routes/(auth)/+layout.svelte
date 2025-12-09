<script lang="ts">
	import { branding } from '$lib/stores/branding';
</script>

<div class="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
	<div class="sm:mx-auto sm:w-full sm:max-w-md">
		{#if $branding.logo_url}
			<img
				class="mx-auto h-16 w-auto"
				src={$branding.logo_url}
				alt={$branding.company_name}
			/>
		{:else}
			<h1 class="text-center text-3xl font-bold text-primary">
				{$branding.company_name}
			</h1>
		{/if}
	</div>

	<div class="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
		<div class="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
			<slot />
		</div>
	</div>

	{#if $branding.footer_text}
		<p class="mt-8 text-center text-sm text-gray-500">
			{$branding.footer_text}
		</p>
	{/if}
</div>
