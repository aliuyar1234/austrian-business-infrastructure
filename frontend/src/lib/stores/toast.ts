import { writable, derived } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
	id: string;
	type: ToastType;
	title: string;
	message?: string;
	duration?: number;
	dismissible?: boolean;
	action?: {
		label: string;
		onClick: () => void;
	};
}

interface ToastState {
	toasts: Toast[];
}

function createToastStore() {
	const { subscribe, update } = writable<ToastState>({ toasts: [] });

	let counter = 0;

	function add(toast: Omit<Toast, 'id'>) {
		const id = `toast-${++counter}`;
		const newToast: Toast = {
			id,
			duration: 5000,
			dismissible: true,
			...toast,
		};

		update((state) => ({
			toasts: [...state.toasts, newToast],
		}));

		// Auto-dismiss after duration
		if (newToast.duration && newToast.duration > 0) {
			setTimeout(() => {
				dismiss(id);
			}, newToast.duration);
		}

		return id;
	}

	function dismiss(id: string) {
		update((state) => ({
			toasts: state.toasts.filter((t) => t.id !== id),
		}));
	}

	function clear() {
		update(() => ({ toasts: [] }));
	}

	// Convenience methods
	function success(title: string, message?: string) {
		return add({ type: 'success', title, message });
	}

	function error(title: string, message?: string) {
		return add({ type: 'error', title, message, duration: 8000 });
	}

	function warning(title: string, message?: string) {
		return add({ type: 'warning', title, message });
	}

	function info(title: string, message?: string) {
		return add({ type: 'info', title, message });
	}

	return {
		subscribe,
		add,
		dismiss,
		clear,
		success,
		error,
		warning,
		info,
	};
}

export const toast = createToastStore();

// Derived store for just the toasts array
export const toasts = derived(toast, ($toast) => $toast.toasts);
