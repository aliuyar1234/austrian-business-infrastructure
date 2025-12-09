<script lang="ts">
	import { page } from '$app/stores';
	import { clsx } from 'clsx';
	import {
		Home,
		Upload,
		FileText,
		CheckCircle,
		ListTodo,
		MessageSquare,
		X
	} from 'lucide-svelte';

	export let open = false;
	export let onClose: () => void;

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: Home },
		{ href: '/upload', label: 'Belege hochladen', icon: Upload },
		{ href: '/documents', label: 'Dokumente', icon: FileText },
		{ href: '/approvals', label: 'Freigaben', icon: CheckCircle },
		{ href: '/tasks', label: 'Aufgaben', icon: ListTodo },
		{ href: '/messages', label: 'Nachrichten', icon: MessageSquare }
	];

	function isActive(href: string) {
		if (href === '/') {
			return $page.url.pathname === '/';
		}
		return $page.url.pathname.startsWith(href);
	}
</script>

<!-- Mobile overlay -->
{#if open}
	<div
		class="fixed inset-0 bg-black/50 z-40 lg:hidden"
		on:click={onClose}
		on:keydown={(e) => e.key === 'Escape' && onClose()}
		role="button"
		tabindex="0"
	/>
{/if}

<!-- Sidebar -->
<aside
	class={clsx(
		'fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-gray-200 transform transition-transform duration-200 ease-in-out',
		'lg:translate-x-0 lg:static lg:z-auto',
		open ? 'translate-x-0' : '-translate-x-full'
	)}
>
	<div class="flex flex-col h-full">
		<!-- Mobile close button -->
		<div class="lg:hidden flex items-center justify-between p-4 border-b border-gray-200">
			<span class="text-lg font-semibold">Menü</span>
			<button
				class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
				on:click={onClose}
			>
				<X class="w-5 h-5" />
			</button>
		</div>

		<!-- Navigation -->
		<nav class="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
			{#each navItems as item}
				<a
					href={item.href}
					class={clsx(
						'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
						isActive(item.href)
							? 'bg-primary/10 text-primary'
							: 'text-gray-700 hover:bg-gray-100'
					)}
					on:click={onClose}
				>
					<svelte:component this={item.icon} class="w-5 h-5" />
					{item.label}
				</a>
			{/each}
		</nav>

		<!-- Footer -->
		<div class="p-4 border-t border-gray-200">
			<p class="text-xs text-gray-500 text-center">
				&copy; {new Date().getFullYear()} Mandantenportal
			</p>
		</div>
	</div>
</aside>
