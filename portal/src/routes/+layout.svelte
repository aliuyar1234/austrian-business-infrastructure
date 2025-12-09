<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { branding } from '$lib/stores/branding';
	import { api } from '$lib/api/client';

	onMount(async () => {
		auth.initialize();

		// Load branding
		try {
			const brandingData = await api.getBranding();
			branding.setBranding(brandingData);
		} catch {
			// Use default branding
		}
	});
</script>

<slot />
