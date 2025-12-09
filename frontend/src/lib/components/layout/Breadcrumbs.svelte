<script lang="ts">
	import { page } from '$app/stores';

	interface Breadcrumb {
		label: string;
		href: string;
	}

	// Generate breadcrumbs from current path
	let breadcrumbs = $derived(() => {
		const pathname = $page.url.pathname;
		const segments = pathname.split('/').filter(Boolean);
		const crumbs: Breadcrumb[] = [{ label: 'Home', href: '/' }];

		let path = '';
		for (const segment of segments) {
			// Skip group folders like (app) or (auth)
			if (segment.startsWith('(') && segment.endsWith(')')) continue;

			path += `/${segment}`;
			const label = getLabel(segment);
			crumbs.push({ label, href: path });
		}

		return crumbs;
	});

	function getLabel(segment: string): string {
		// Convert URL segments to readable labels
		const labels: Record<string, string> = {
			accounts: 'Accounts',
			documents: 'Documents',
			uva: 'UVA',
			invoices: 'Invoices',
			sepa: 'SEPA',
			firmenbuch: 'Firmenbuch',
			calendar: 'Calendar',
			team: 'Team',
			settings: 'Settings',
			new: 'New',
			login: 'Login',
			register: 'Register',
			'forgot-password': 'Reset Password',
		};

		// Check if it looks like an ID (UUID or similar)
		if (segment.match(/^[a-f0-9-]{8,}$/i)) {
			return 'Details';
		}

		return labels[segment] || segment.charAt(0).toUpperCase() + segment.slice(1);
	}
</script>

{#if breadcrumbs().length > 1}
	<nav aria-label="Breadcrumb" class="flex items-center gap-2 text-sm text-[var(--color-ink-muted)]">
		{#each breadcrumbs() as crumb, i}
			{#if i > 0}
				<svg class="w-4 h-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="m9 18 6-6-6-6"/>
				</svg>
			{/if}

			{#if i === breadcrumbs().length - 1}
				<!-- Current page (not a link) -->
				<span class="text-[var(--color-ink)] font-medium truncate max-w-[200px]">
					{crumb.label}
				</span>
			{:else}
				<a
					href={crumb.href}
					class="hover:text-[var(--color-ink)] transition-colors truncate max-w-[200px]"
				>
					{crumb.label}
				</a>
			{/if}
		{/each}
	</nav>
{/if}
