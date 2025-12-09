<script lang="ts">
	import { onMount } from 'svelte';
	import { ListTodo, CheckCircle, Clock, AlertTriangle } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { api } from '$lib/api/client';

	let tasks: any[] = [];
	let loading = true;
	let total = 0;
	let filter: 'open' | 'completed' | 'all' = 'open';

	onMount(async () => {
		await loadTasks();
	});

	async function loadTasks() {
		loading = true;
		try {
			const status = filter === 'all' ? undefined : filter;
			const result = await api.getTasks({ status, limit: 50 });
			tasks = result.tasks || [];
			total = result.total;
		} catch (e) {
			console.error('Failed to load tasks:', e);
		} finally {
			loading = false;
		}
	}

	function formatDate(date: string): string {
		return new Date(date).toLocaleDateString('de-AT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric'
		});
	}

	function isOverdue(dueDate: string): boolean {
		return new Date(dueDate) < new Date();
	}

	function getPriorityBadge(priority: string) {
		switch (priority) {
			case 'high':
				return { variant: 'danger' as const, label: 'Hoch' };
			case 'medium':
				return { variant: 'warning' as const, label: 'Mittel' };
			case 'low':
				return { variant: 'default' as const, label: 'Niedrig' };
			default:
				return { variant: 'default' as const, label: priority };
		}
	}

	$: if (filter) loadTasks();
</script>

<svelte:head>
	<title>Aufgaben | Mandantenportal</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold text-gray-900">Aufgaben</h1>

		<div class="flex gap-2">
			<button
				class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
				class:bg-primary={filter === 'open'}
				class:text-white={filter === 'open'}
				class:bg-gray-100={filter !== 'open'}
				class:text-gray-700={filter !== 'open'}
				on:click={() => (filter = 'open')}
			>
				Offen
			</button>
			<button
				class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
				class:bg-primary={filter === 'completed'}
				class:text-white={filter === 'completed'}
				class:bg-gray-100={filter !== 'completed'}
				class:text-gray-700={filter !== 'completed'}
				on:click={() => (filter = 'completed')}
			>
				Erledigt
			</button>
			<button
				class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
				class:bg-primary={filter === 'all'}
				class:text-white={filter === 'all'}
				class:bg-gray-100={filter !== 'all'}
				class:text-gray-700={filter !== 'all'}
				on:click={() => (filter = 'all')}
			>
				Alle
			</button>
		</div>
	</div>

	{#if loading}
		<div class="space-y-4">
			{#each [1, 2, 3] as _}
				<Card>
					<div class="h-20 bg-gray-100 rounded animate-pulse"></div>
				</Card>
			{/each}
		</div>
	{:else if tasks.length === 0}
		<Card class="text-center py-12">
			<ListTodo class="w-16 h-16 mx-auto mb-4 text-gray-300" />
			<h2 class="text-xl font-semibold text-gray-900 mb-2">
				{filter === 'open' ? 'Keine offenen Aufgaben' : 'Keine Aufgaben'}
			</h2>
			<p class="text-gray-600">
				{filter === 'open'
					? 'Alle Aufgaben wurden erledigt.'
					: 'Es wurden noch keine Aufgaben zugewiesen.'}
			</p>
		</Card>
	{:else}
		<div class="space-y-3">
			{#each tasks as task}
				{@const priority = getPriorityBadge(task.priority)}
				{@const overdue = task.due_date && task.status === 'open' && isOverdue(task.due_date)}
				<Card padding="none">
					<a
						href="/tasks/{task.id}"
						class="flex items-center gap-4 p-4 hover:bg-gray-50 transition-colors"
					>
						<div class="p-3 rounded-lg" class:bg-green-100={task.status === 'completed'} class:bg-red-100={overdue} class:bg-gray-100={!overdue && task.status !== 'completed'}>
							{#if task.status === 'completed'}
								<CheckCircle class="w-6 h-6 text-green-600" />
							{:else if overdue}
								<AlertTriangle class="w-6 h-6 text-red-600" />
							{:else}
								<Clock class="w-6 h-6 text-gray-600" />
							{/if}
						</div>

						<div class="flex-1 min-w-0">
							<p class="font-medium text-gray-900 truncate">
								{task.title}
							</p>
							{#if task.description}
								<p class="text-sm text-gray-500 truncate">
									{task.description}
								</p>
							{/if}
							{#if task.due_date}
								<p class="text-sm mt-1" class:text-red-600={overdue} class:text-gray-500={!overdue}>
									Fällig: {formatDate(task.due_date)}
									{#if overdue}
										<span class="font-medium">(überfällig)</span>
									{/if}
								</p>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							<Badge variant={priority.variant}>{priority.label}</Badge>
						</div>
					</a>
				</Card>
			{/each}
		</div>
	{/if}
</div>
