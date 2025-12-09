import { writable, derived } from 'svelte/store';
import { api } from '$lib/api/client';
import { toast } from './toast';

export type AccountType = 'finanzonline' | 'elda' | 'firmenbuch';
export type AccountStatus = 'active' | 'syncing' | 'error' | 'pending';

export interface Account {
	id: string;
	name: string;
	type: AccountType;
	status: AccountStatus;
	teilnehmerId?: string; // FO
	benutzerId?: string; // FO
	dienstgeberId?: string; // ELDA
	lastSync?: Date;
	lastError?: string;
	documentCount: number;
	tags: string[];
	createdAt: Date;
	updatedAt: Date;
}

export interface CreateAccountRequest {
	name: string;
	type: AccountType;
	// FinanzOnline credentials
	teilnehmerId?: string;
	benutzerId?: string;
	pin?: string;
	// ELDA credentials
	dienstgeberId?: string;
	eldaPassword?: string;
	// Firmenbuch credentials
	username?: string;
	password?: string;
	// Tags
	tags?: string[];
}

export interface UpdateAccountRequest {
	name?: string;
	pin?: string;
	eldaPassword?: string;
	password?: string;
	tags?: string[];
}

interface AccountsState {
	accounts: Account[];
	loading: boolean;
	error: string | null;
	selectedId: string | null;
}

function createAccountsStore() {
	const { subscribe, set, update } = writable<AccountsState>({
		accounts: [],
		loading: false,
		error: null,
		selectedId: null,
	});

	async function load() {
		update((s) => ({ ...s, loading: true, error: null }));
		try {
			const data = await api.get<Account[]>('/api/v1/accounts');
			update((s) => ({
				...s,
				accounts: data.map((a) => ({
					...a,
					lastSync: a.lastSync ? new Date(a.lastSync) : undefined,
					createdAt: new Date(a.createdAt),
					updatedAt: new Date(a.updatedAt),
				})),
				loading: false,
			}));
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load accounts';
			update((s) => ({ ...s, loading: false, error: message }));
		}
	}

	async function create(request: CreateAccountRequest) {
		update((s) => ({ ...s, loading: true, error: null }));
		try {
			const account = await api.post<Account>('/api/v1/accounts', request);
			update((s) => ({
				...s,
				accounts: [
					...s.accounts,
					{
						...account,
						lastSync: account.lastSync ? new Date(account.lastSync) : undefined,
						createdAt: new Date(account.createdAt),
						updatedAt: new Date(account.updatedAt),
					},
				],
				loading: false,
			}));
			toast.success('Account created', `${request.name} has been added successfully`);
			return { success: true, account };
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to create account';
			update((s) => ({ ...s, loading: false, error: message }));
			toast.error('Failed to create account', message);
			return { success: false, error: message };
		}
	}

	async function updateAccount(id: string, request: UpdateAccountRequest) {
		update((s) => ({ ...s, loading: true, error: null }));
		try {
			const account = await api.put<Account>(`/api/v1/accounts/${id}`, request);
			update((s) => ({
				...s,
				accounts: s.accounts.map((a) =>
					a.id === id
						? {
								...account,
								lastSync: account.lastSync ? new Date(account.lastSync) : undefined,
								createdAt: new Date(account.createdAt),
								updatedAt: new Date(account.updatedAt),
						  }
						: a
				),
				loading: false,
			}));
			toast.success('Account updated', 'Changes have been saved');
			return { success: true };
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to update account';
			update((s) => ({ ...s, loading: false, error: message }));
			toast.error('Failed to update account', message);
			return { success: false, error: message };
		}
	}

	async function remove(id: string) {
		update((s) => ({ ...s, loading: true, error: null }));
		try {
			await api.delete(`/api/v1/accounts/${id}`);
			update((s) => ({
				...s,
				accounts: s.accounts.filter((a) => a.id !== id),
				loading: false,
			}));
			toast.success('Account deleted', 'The account has been removed');
			return { success: true };
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to delete account';
			update((s) => ({ ...s, loading: false, error: message }));
			toast.error('Failed to delete account', message);
			return { success: false, error: message };
		}
	}

	async function sync(id: string) {
		// Update status to syncing
		update((s) => ({
			...s,
			accounts: s.accounts.map((a) => (a.id === id ? { ...a, status: 'syncing' as AccountStatus } : a)),
		}));

		try {
			await api.post(`/api/v1/accounts/${id}/sync`);
			toast.info('Sync started', 'Fetching documents from databox...');
			return { success: true };
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to sync account';
			update((s) => ({
				...s,
				accounts: s.accounts.map((a) =>
					a.id === id ? { ...a, status: 'error' as AccountStatus, lastError: message } : a
				),
			}));
			toast.error('Sync failed', message);
			return { success: false, error: message };
		}
	}

	async function testConnection(id: string) {
		try {
			const result = await api.post<{ success: boolean; message: string }>(`/api/v1/accounts/${id}/test`);
			if (result.success) {
				toast.success('Connection successful', result.message);
			} else {
				toast.error('Connection failed', result.message);
			}
			return result;
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Connection test failed';
			toast.error('Connection test failed', message);
			return { success: false, message };
		}
	}

	function select(id: string | null) {
		update((s) => ({ ...s, selectedId: id }));
	}

	return {
		subscribe,
		load,
		create,
		update: updateAccount,
		remove,
		sync,
		testConnection,
		select,
	};
}

export const accounts = createAccountsStore();

// Derived stores
export const accountsList = derived(accounts, ($accounts) => $accounts.accounts);
export const accountsLoading = derived(accounts, ($accounts) => $accounts.loading);
export const accountsError = derived(accounts, ($accounts) => $accounts.error);
export const selectedAccount = derived(accounts, ($accounts) => {
	if (!$accounts.selectedId) return null;
	return $accounts.accounts.find((a) => a.id === $accounts.selectedId) || null;
});

// Helper functions
export function getAccountTypeLabel(type: AccountType): string {
	switch (type) {
		case 'finanzonline':
			return 'FinanzOnline';
		case 'elda':
			return 'ELDA';
		case 'firmenbuch':
			return 'Firmenbuch';
	}
}

export function getAccountStatusLabel(status: AccountStatus): string {
	switch (status) {
		case 'active':
			return 'Active';
		case 'syncing':
			return 'Syncing';
		case 'error':
			return 'Error';
		case 'pending':
			return 'Pending';
	}
}
