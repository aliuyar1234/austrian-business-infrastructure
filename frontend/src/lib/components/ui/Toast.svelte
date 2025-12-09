<script lang="ts">
	import { cn } from '$lib/utils';
	import { toasts, toast, type Toast, type ToastType } from '$lib/stores/toast';
	import { flip } from 'svelte/animate';

	function getToastStyles(type: ToastType) {
		switch (type) {
			case 'success':
				return {
					bg: 'bg-[var(--color-success-muted)]',
					border: 'border-[var(--color-success)]/20',
					icon: 'text-[var(--color-success)]',
					iconPath: '<polyline points="20 6 9 17 4 12"/>'
				};
			case 'error':
				return {
					bg: 'bg-[var(--color-error-muted)]',
					border: 'border-[var(--color-error)]/20',
					icon: 'text-[var(--color-error)]',
					iconPath: '<circle cx="12" cy="12" r="10"/><line x1="15" x2="9" y1="9" y2="15"/><line x1="9" x2="15" y1="9" y2="15"/>'
				};
			case 'warning':
				return {
					bg: 'bg-[var(--color-warning-muted)]',
					border: 'border-[var(--color-warning)]/20',
					icon: 'text-[var(--color-warning)]',
					iconPath: '<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><path d="M12 9v4"/><path d="M12 17h.01"/>'
				};
			case 'info':
				return {
					bg: 'bg-[var(--color-info-muted)]',
					border: 'border-[var(--color-info)]/20',
					icon: 'text-[var(--color-info)]',
					iconPath: '<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>'
				};
		}
	}
</script>

<!-- Toast container -->
<div
	class="fixed bottom-4 right-4 z-[var(--z-toast)] flex flex-col gap-2 pointer-events-none"
	aria-live="polite"
	aria-atomic="true"
>
	{#each $toasts as t (t.id)}
		{@const styles = getToastStyles(t.type)}
		<div
			animate:flip={{ duration: 200 }}
			class={cn(
				'pointer-events-auto w-80 sm:w-96',
				'flex items-start gap-3 p-4',
				'rounded-lg border shadow-lg',
				'animate-in',
				styles.bg,
				styles.border
			)}
			role="alert"
		>
			<!-- Icon -->
			<div class={cn('flex-shrink-0 w-5 h-5', styles.icon)}>
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					class="w-5 h-5"
				>
					{@html styles.iconPath}
				</svg>
			</div>

			<!-- Content -->
			<div class="flex-1 min-w-0">
				<p class="text-sm font-medium text-[var(--color-ink)]">
					{t.title}
				</p>
				{#if t.message}
					<p class="mt-0.5 text-sm text-[var(--color-ink-secondary)]">
						{t.message}
					</p>
				{/if}
				{#if t.action}
					<button
						onclick={t.action.onClick}
						class="mt-2 text-sm font-medium text-[var(--color-accent)] hover:underline"
					>
						{t.action.label}
					</button>
				{/if}
			</div>

			<!-- Dismiss button -->
			{#if t.dismissible}
				<button
					onclick={() => toast.dismiss(t.id)}
					class="flex-shrink-0 w-6 h-6 flex items-center justify-center rounded hover:bg-black/5 text-[var(--color-ink-muted)] hover:text-[var(--color-ink)]"
					aria-label="Dismiss"
				>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M18 6 6 18M6 6l12 12"/>
					</svg>
				</button>
			{/if}
		</div>
	{/each}
</div>
