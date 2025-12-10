<script lang="ts">
	import { page } from '$app/stores';
	import { cn } from '$lib/utils';
	import { user } from '$lib/stores/auth';

	// Lucide icons as inline SVGs for tree-shaking
	const icons = {
		home: `<path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/>`,
		folder: `<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>`,
		users: `<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>`,
		building: `<rect width="16" height="20" x="4" y="2" rx="2" ry="2"/><path d="M9 22v-4h6v4"/><path d="M8 6h.01"/><path d="M16 6h.01"/><path d="M12 6h.01"/><path d="M12 10h.01"/><path d="M12 14h.01"/><path d="M16 10h.01"/><path d="M16 14h.01"/><path d="M8 10h.01"/><path d="M8 14h.01"/>`,
		fileText: `<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" x2="8" y1="13" y2="13"/><line x1="16" x2="8" y1="17" y2="17"/><polyline points="10 9 9 9 8 9"/>`,
		calculator: `<rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/>`,
		receipt: `<path d="M4 2v20l2-1 2 1 2-1 2 1 2-1 2 1 2-1 2 1V2l-2 1-2-1-2 1-2-1-2 1-2-1-2 1Z"/><path d="M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8"/><path d="M12 17.5v-11"/>`,
		creditCard: `<rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/>`,
		search: `<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>`,
		calendar: `<path d="M8 2v4"/><path d="M16 2v4"/><rect width="18" height="18" x="3" y="4" rx="2"/><path d="M3 10h18"/>`,
		settings: `<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>`,
		sparkles: `<path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/><path d="M5 3v4"/><path d="M19 17v4"/><path d="M3 5h4"/><path d="M17 19h4"/>`,
		clipboardList: `<rect width="8" height="4" x="8" y="2" rx="1" ry="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="M12 11h4"/><path d="M12 16h4"/><path d="M8 11h.01"/><path d="M8 16h.01"/>`,
		briefcase: `<rect width="20" height="14" x="2" y="7" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>`,
		euro: `<path d="M4 10h12"/><path d="M4 14h9"/><path d="M19 6a7.7 7.7 0 0 0-5.2-2A7.9 7.9 0 0 0 6 12c0 4.4 3.5 8 7.8 8 2 0 3.8-.8 5.2-2"/>`
	};

	interface NavItem {
		label: string;
		href: string;
		icon: keyof typeof icons;
		badge?: string;
	}

	interface NavGroup {
		label: string;
		items: NavItem[];
	}

	const navigation: NavGroup[] = [
		{
			label: 'Overview',
			items: [
				{ label: 'Dashboard', href: '/', icon: 'home' },
				{ label: 'Documents', href: '/documents', icon: 'folder' },
				{ label: 'Accounts', href: '/accounts', icon: 'building' }
			]
		},
		{
			label: 'Förderungen',
			items: [
				{ label: 'Förderungssuche', href: '/foerderungen/suche', icon: 'sparkles' },
				{ label: 'Datenbank', href: '/foerderungen', icon: 'euro' },
				{ label: 'Profile', href: '/profile', icon: 'briefcase' },
				{ label: 'Anträge', href: '/antraege', icon: 'clipboardList' }
			]
		},
		{
			label: 'Tax & Finance',
			items: [
				{ label: 'UVA', href: '/uva', icon: 'calculator' },
				{ label: 'E-Rechnung', href: '/invoices', icon: 'receipt' },
				{ label: 'SEPA', href: '/sepa', icon: 'creditCard' },
				{ label: 'Firmenbuch', href: '/firmenbuch', icon: 'search' }
			]
		},
		{
			label: 'Organization',
			items: [
				{ label: 'Calendar', href: '/calendar', icon: 'calendar' },
				{ label: 'Team', href: '/team', icon: 'users' },
				{ label: 'Settings', href: '/settings', icon: 'settings' }
			]
		}
	];

	function isActive(href: string, pathname: string): boolean {
		if (href === '/') return pathname === '/';
		return pathname.startsWith(href);
	}
</script>

<aside class="w-64 h-screen bg-[var(--color-paper-elevated)] border-r border-black/6 flex flex-col">
	<!-- Logo -->
	<div class="h-16 px-5 flex items-center border-b border-black/6">
		<a href="/" class="flex items-center gap-3 group">
			<!-- Austrian red accent square -->
			<div class="w-8 h-8 rounded-md bg-[var(--color-accent)] flex items-center justify-center">
				<span class="text-white font-bold text-sm">ABI</span>
			</div>
			<div class="flex flex-col">
				<span class="text-sm font-semibold text-[var(--color-ink)] leading-tight">
					Austrian Business
				</span>
				<span class="text-xs text-[var(--color-ink-muted)] leading-tight">
					Infrastructure
				</span>
			</div>
		</a>
	</div>

	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto py-4 px-3">
		{#each navigation as group, groupIndex}
			<div class={cn(groupIndex > 0 && 'mt-6')}>
				<h3 class="px-3 mb-2 text-xs font-medium text-[var(--color-ink-muted)] uppercase tracking-wider">
					{group.label}
				</h3>
				<ul class="space-y-0.5">
					{#each group.items as item}
						{@const active = isActive(item.href, $page.url.pathname)}
						<li>
							<a
								href={item.href}
								class={cn(
									'nav-item',
									active && 'active'
								)}
							>
								<svg
									class="w-5 h-5 flex-shrink-0"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="1.5"
									stroke-linecap="round"
									stroke-linejoin="round"
								>
									{@html icons[item.icon]}
								</svg>
								<span>{item.label}</span>
								{#if item.badge}
									<span class="ml-auto badge badge-info text-[10px] px-1.5 py-0.5">
										{item.badge}
									</span>
								{/if}
							</a>
						</li>
					{/each}
				</ul>
			</div>
		{/each}
	</nav>

	<!-- User section -->
	{#if $user}
		<div class="p-3 border-t border-black/6">
			<a
				href="/settings"
				class="flex items-center gap-3 p-2 rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors"
			>
				<div class="w-9 h-9 rounded-full bg-[var(--color-accent-muted)] flex items-center justify-center">
					<span class="text-sm font-medium text-[var(--color-accent)]">
						{$user.name.split(' ').map(n => n[0]).join('').slice(0, 2).toUpperCase()}
					</span>
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-sm font-medium text-[var(--color-ink)] truncate">
						{$user.name}
					</p>
					<p class="text-xs text-[var(--color-ink-muted)] truncate">
						{$user.tenantName}
					</p>
				</div>
			</a>
		</div>
	{/if}
</aside>
