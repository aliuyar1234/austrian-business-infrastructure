<script lang="ts">
	import { formatDate } from '$lib/utils';
	import { toast } from '$lib/stores/toast';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	type MemberRole = 'owner' | 'admin' | 'member';
	type InvitationStatus = 'pending' | 'accepted' | 'expired';

	interface TeamMember {
		id: string;
		name: string;
		email: string;
		role: MemberRole;
		avatar?: string;
		joinedAt: Date;
	}

	interface Invitation {
		id: string;
		email: string;
		role: MemberRole;
		status: InvitationStatus;
		invitedAt: Date;
		expiresAt: Date;
	}

	// Mock data
	let members = $state<TeamMember[]>([
		{
			id: '1',
			name: 'Johann Mustermann',
			email: 'johann@musterfirma.at',
			role: 'owner',
			joinedAt: new Date(Date.now() - 180 * 86400000),
		},
		{
			id: '2',
			name: 'Maria Musterfrau',
			email: 'maria@musterfirma.at',
			role: 'admin',
			joinedAt: new Date(Date.now() - 90 * 86400000),
		},
		{
			id: '3',
			name: 'Thomas Tester',
			email: 'thomas@musterfirma.at',
			role: 'member',
			joinedAt: new Date(Date.now() - 30 * 86400000),
		},
	]);

	let invitations = $state<Invitation[]>([
		{
			id: '1',
			email: 'new.member@example.com',
			role: 'member',
			status: 'pending',
			invitedAt: new Date(Date.now() - 2 * 86400000),
			expiresAt: new Date(Date.now() + 5 * 86400000),
		},
	]);

	let showInviteModal = $state(false);
	let inviteEmail = $state('');
	let inviteRole = $state<MemberRole>('member');
	let isInviting = $state(false);

	function getRoleLabel(role: MemberRole): string {
		switch (role) {
			case 'owner': return 'Owner';
			case 'admin': return 'Admin';
			case 'member': return 'Member';
		}
	}

	function getRoleVariant(role: MemberRole): 'default' | 'warning' | 'info' {
		switch (role) {
			case 'owner': return 'warning';
			case 'admin': return 'info';
			case 'member': return 'default';
		}
	}

	function getInitials(name: string): string {
		return name.split(' ').map(n => n[0]).join('').slice(0, 2).toUpperCase();
	}

	async function handleInvite() {
		if (!inviteEmail.trim() || !inviteEmail.includes('@')) {
			toast.error('Invalid email', 'Please enter a valid email address');
			return;
		}

		isInviting = true;
		await new Promise(r => setTimeout(r, 1000));

		invitations = [
			...invitations,
			{
				id: String(Date.now()),
				email: inviteEmail,
				role: inviteRole,
				status: 'pending',
				invitedAt: new Date(),
				expiresAt: new Date(Date.now() + 7 * 86400000),
			}
		];

		isInviting = false;
		showInviteModal = false;
		inviteEmail = '';
		inviteRole = 'member';
		toast.success('Invitation sent', `Invitation sent to ${inviteEmail}`);
	}

	async function cancelInvitation(id: string) {
		invitations = invitations.filter(i => i.id !== id);
		toast.info('Invitation cancelled');
	}

	async function resendInvitation(id: string) {
		toast.success('Invitation resent');
	}

	async function changeRole(memberId: string, newRole: MemberRole) {
		members = members.map(m => m.id === memberId ? { ...m, role: newRole } : m);
		toast.success('Role updated');
	}

	async function removeMember(id: string) {
		members = members.filter(m => m.id !== id);
		toast.success('Member removed');
	}
</script>

<svelte:head>
	<title>Team - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<h1 class="text-xl font-semibold text-[var(--color-ink)]">Team</h1>
			<p class="text-sm text-[var(--color-ink-muted)]">
				Manage team members and their permissions
			</p>
		</div>
		<Button onclick={() => { showInviteModal = true; }}>
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/>
				<circle cx="9" cy="7" r="4"/>
				<line x1="19" x2="19" y1="8" y2="14"/>
				<line x1="22" x2="16" y1="11" y2="11"/>
			</svg>
			Invite member
		</Button>
	</div>

	<!-- Team members -->
	<Card padding="none">
		<div class="p-4 border-b border-black/6">
			<h2 class="font-semibold text-[var(--color-ink)]">Team Members ({members.length})</h2>
		</div>
		<div class="divide-y divide-black/4">
			{#each members as member}
				<div class="flex items-center justify-between p-4">
					<div class="flex items-center gap-4">
						<div class="w-10 h-10 rounded-full bg-[var(--color-accent-muted)] flex items-center justify-center">
							<span class="text-sm font-medium text-[var(--color-accent)]">
								{getInitials(member.name)}
							</span>
						</div>
						<div>
							<div class="flex items-center gap-2">
								<p class="font-medium text-[var(--color-ink)]">{member.name}</p>
								<Badge variant={getRoleVariant(member.role)} size="sm">
									{getRoleLabel(member.role)}
								</Badge>
							</div>
							<p class="text-sm text-[var(--color-ink-muted)]">{member.email}</p>
						</div>
					</div>

					<div class="flex items-center gap-3">
						<span class="text-xs text-[var(--color-ink-muted)] hidden sm:block">
							Joined {formatDate(member.joinedAt)}
						</span>
						{#if member.role !== 'owner'}
							<select
								value={member.role}
								onchange={(e) => changeRole(member.id, (e.target as HTMLSelectElement).value as MemberRole)}
								class="input h-8 text-sm w-28"
							>
								<option value="admin">Admin</option>
								<option value="member">Member</option>
							</select>
							<Button variant="ghost" size="sm" onclick={() => removeMember(member.id)}>
								<svg class="w-4 h-4 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M3 6h18M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
								</svg>
							</Button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	</Card>

	<!-- Pending invitations -->
	{#if invitations.length > 0}
		<Card padding="none">
			<div class="p-4 border-b border-black/6">
				<h2 class="font-semibold text-[var(--color-ink)]">Pending Invitations ({invitations.length})</h2>
			</div>
			<div class="divide-y divide-black/4">
				{#each invitations as invitation}
					<div class="flex items-center justify-between p-4">
						<div class="flex items-center gap-4">
							<div class="w-10 h-10 rounded-full bg-[var(--color-paper-inset)] flex items-center justify-center">
								<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
									<polyline points="22,6 12,13 2,6"/>
								</svg>
							</div>
							<div>
								<p class="font-medium text-[var(--color-ink)]">{invitation.email}</p>
								<div class="flex items-center gap-2 mt-0.5">
									<Badge variant={getRoleVariant(invitation.role)} size="sm">
										{getRoleLabel(invitation.role)}
									</Badge>
									<span class="text-xs text-[var(--color-ink-muted)]">
										Expires {formatDate(invitation.expiresAt)}
									</span>
								</div>
							</div>
						</div>

						<div class="flex items-center gap-2">
							<Button variant="ghost" size="sm" onclick={() => resendInvitation(invitation.id)}>
								<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
									<path d="M3 3v5h5"/>
								</svg>
								Resend
							</Button>
							<Button variant="ghost" size="sm" onclick={() => cancelInvitation(invitation.id)}>
								<svg class="w-4 h-4 text-[var(--color-error)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M18 6 6 18M6 6l12 12"/>
								</svg>
							</Button>
						</div>
					</div>
				{/each}
			</div>
		</Card>
	{/if}
</div>

<!-- Invite modal -->
{#if showInviteModal}
	<div class="fixed inset-0 bg-black/50 backdrop-blur-sm z-[var(--z-modal)] flex items-center justify-center p-4">
		<div class="bg-[var(--color-paper-elevated)] rounded-xl shadow-2xl max-w-md w-full p-6 animate-in">
			<h3 class="text-lg font-semibold text-[var(--color-ink)]">Invite team member</h3>
			<p class="text-sm text-[var(--color-ink-muted)] mt-1">
				Send an invitation to join your team.
			</p>

			<form onsubmit={(e) => { e.preventDefault(); handleInvite(); }} class="mt-6 space-y-4">
				<div>
					<label for="invite-email" class="label">Email address</label>
					<Input type="email" id="invite-email" bind:value={inviteEmail} placeholder="colleague@company.at" />
				</div>
				<div>
					<label for="invite-role" class="label">Role</label>
					<select id="invite-role" bind:value={inviteRole} class="input h-10">
						<option value="member">Member - Can view and create content</option>
						<option value="admin">Admin - Can manage team and settings</option>
					</select>
				</div>

				<div class="flex justify-end gap-2 pt-2">
					<Button variant="secondary" type="button" onclick={() => { showInviteModal = false; }}>Cancel</Button>
					<Button type="submit" loading={isInviting}>Send invitation</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
