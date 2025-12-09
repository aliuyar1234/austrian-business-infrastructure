<script lang="ts">
	import { user } from '$lib/stores/auth';
	import { toast } from '$lib/stores/toast';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	// Profile form
	let profileName = $state($user?.name ?? '');
	let profileEmail = $state($user?.email ?? '');
	let isSavingProfile = $state(false);

	// Password form
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let isChangingPassword = $state(false);

	// Notification settings
	let emailNotifications = $state(true);
	let browserNotifications = $state(true);
	let soundEnabled = $state(true);

	// 2FA
	let twoFactorEnabled = $state(false);
	let isEnabling2FA = $state(false);

	async function saveProfile() {
		isSavingProfile = true;
		await new Promise(r => setTimeout(r, 1000));
		isSavingProfile = false;
		toast.success('Profile updated', 'Your profile has been saved');
	}

	async function changePassword() {
		if (newPassword !== confirmPassword) {
			toast.error('Passwords do not match');
			return;
		}
		if (newPassword.length < 8) {
			toast.error('Password too short', 'Password must be at least 8 characters');
			return;
		}

		isChangingPassword = true;
		await new Promise(r => setTimeout(r, 1000));
		isChangingPassword = false;
		currentPassword = '';
		newPassword = '';
		confirmPassword = '';
		toast.success('Password changed', 'Your password has been updated');
	}

	async function toggle2FA() {
		isEnabling2FA = true;
		await new Promise(r => setTimeout(r, 1000));
		twoFactorEnabled = !twoFactorEnabled;
		isEnabling2FA = false;
		toast.success(
			twoFactorEnabled ? '2FA enabled' : '2FA disabled',
			twoFactorEnabled ? 'Your account is now more secure' : 'Two-factor authentication has been disabled'
		);
	}

	function getInitials(name: string): string {
		return name.split(' ').map(n => n[0]).join('').slice(0, 2).toUpperCase();
	}
</script>

<svelte:head>
	<title>Settings - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-3xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div>
		<h1 class="text-xl font-semibold text-[var(--color-ink)]">Settings</h1>
		<p class="text-sm text-[var(--color-ink-muted)]">
			Manage your account and preferences
		</p>
	</div>

	<!-- Profile -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Profile</h2>
		<form onsubmit={(e) => { e.preventDefault(); saveProfile(); }} class="space-y-4">
			<div class="flex items-center gap-4 mb-6">
				<div class="w-16 h-16 rounded-full bg-[var(--color-accent-muted)] flex items-center justify-center">
					<span class="text-xl font-medium text-[var(--color-accent)]">
						{$user ? getInitials($user.name) : '?'}
					</span>
				</div>
				<div>
					<p class="font-medium text-[var(--color-ink)]">{$user?.name ?? 'Unknown'}</p>
					<p class="text-sm text-[var(--color-ink-muted)]">{$user?.tenantName ?? 'No organization'}</p>
				</div>
			</div>

			<div class="grid sm:grid-cols-2 gap-4">
				<div>
					<label for="profile-name" class="label">Full name</label>
					<Input type="text" id="profile-name" bind:value={profileName} />
				</div>
				<div>
					<label for="profile-email" class="label">Email address</label>
					<Input type="email" id="profile-email" bind:value={profileEmail} />
					<p class="mt-1 text-xs text-[var(--color-ink-muted)]">Used for login and notifications</p>
				</div>
			</div>

			<div class="flex justify-end pt-2">
				<Button type="submit" loading={isSavingProfile}>Save changes</Button>
			</div>
		</form>
	</Card>

	<!-- Password -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Change Password</h2>
		<form onsubmit={(e) => { e.preventDefault(); changePassword(); }} class="space-y-4">
			<div>
				<label for="current-password" class="label">Current password</label>
				<Input type="password" id="current-password" bind:value={currentPassword} />
			</div>
			<div class="grid sm:grid-cols-2 gap-4">
				<div>
					<label for="new-password" class="label">New password</label>
					<Input type="password" id="new-password" bind:value={newPassword} />
				</div>
				<div>
					<label for="confirm-password" class="label">Confirm new password</label>
					<Input type="password" id="confirm-password" bind:value={confirmPassword} />
				</div>
			</div>

			<div class="flex justify-end pt-2">
				<Button type="submit" loading={isChangingPassword}>Change password</Button>
			</div>
		</form>
	</Card>

	<!-- Two-factor authentication -->
	<Card>
		<div class="flex items-start justify-between">
			<div>
				<h2 class="font-semibold text-[var(--color-ink)]">Two-Factor Authentication</h2>
				<p class="text-sm text-[var(--color-ink-muted)] mt-1">
					Add an extra layer of security to your account
				</p>
			</div>
			<div class="flex items-center gap-3">
				{#if twoFactorEnabled}
					<span class="text-sm text-[var(--color-success)] font-medium">Enabled</span>
				{/if}
				<Button
					variant={twoFactorEnabled ? 'secondary' : 'primary'}
					onclick={toggle2FA}
					loading={isEnabling2FA}
				>
					{twoFactorEnabled ? 'Disable' : 'Enable'} 2FA
				</Button>
			</div>
		</div>

		{#if twoFactorEnabled}
			<div class="mt-4 p-4 rounded-lg bg-[var(--color-success-muted)]">
				<div class="flex gap-3">
					<svg class="w-5 h-5 text-[var(--color-success)] flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
						<path d="m9 12 2 2 4-4"/>
					</svg>
					<div class="text-sm text-[var(--color-success)]">
						<p class="font-medium">Two-factor authentication is active</p>
						<p class="mt-0.5 opacity-80">Your account is protected with an authenticator app</p>
					</div>
				</div>
			</div>
		{/if}
	</Card>

	<!-- Notifications -->
	<Card>
		<h2 class="font-semibold text-[var(--color-ink)] mb-4">Notifications</h2>
		<div class="space-y-4">
			<label class="flex items-center justify-between cursor-pointer">
				<div>
					<p class="font-medium text-[var(--color-ink)]">Email notifications</p>
					<p class="text-sm text-[var(--color-ink-muted)]">Receive emails for new documents and deadlines</p>
				</div>
				<input
					type="checkbox"
					bind:checked={emailNotifications}
					class="w-5 h-5 rounded text-[var(--color-accent)] focus:ring-[var(--color-accent)]"
				/>
			</label>

			<label class="flex items-center justify-between cursor-pointer">
				<div>
					<p class="font-medium text-[var(--color-ink)]">Browser notifications</p>
					<p class="text-sm text-[var(--color-ink-muted)]">Get notified in your browser when new documents arrive</p>
				</div>
				<input
					type="checkbox"
					bind:checked={browserNotifications}
					class="w-5 h-5 rounded text-[var(--color-accent)] focus:ring-[var(--color-accent)]"
				/>
			</label>

			<label class="flex items-center justify-between cursor-pointer">
				<div>
					<p class="font-medium text-[var(--color-ink)]">Notification sound</p>
					<p class="text-sm text-[var(--color-ink-muted)]">Play a sound when notifications arrive</p>
				</div>
				<input
					type="checkbox"
					bind:checked={soundEnabled}
					class="w-5 h-5 rounded text-[var(--color-accent)] focus:ring-[var(--color-accent)]"
				/>
			</label>
		</div>
	</Card>

	<!-- Danger zone -->
	<Card padding="none">
		<div class="p-4 border-b border-black/6">
			<h2 class="font-semibold text-[var(--color-error)]">Danger Zone</h2>
		</div>
		<div class="p-4">
			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium text-[var(--color-ink)]">Delete account</p>
					<p class="text-sm text-[var(--color-ink-muted)]">Permanently delete your account and all data</p>
				</div>
				<Button variant="danger">Delete account</Button>
			</div>
		</div>
	</Card>
</div>
