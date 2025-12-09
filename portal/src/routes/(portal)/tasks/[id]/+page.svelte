<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { ArrowLeft, ListTodo, CheckCircle, FileText, Upload } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { api } from '$lib/api/client';

	let task: any = null;
	let loading = true;
	let error = '';
	let completing = false;

	onMount(async () => {
		const id = $page.params.id;

		try {
			task = await api.getTask(id);
		} catch (e: any) {
			error = e.message || 'Aufgabe nicht gefunden';
		} finally {
			loading = false;
		}
	});

	async function handleComplete() {
		completing = true;
		try {
			await api.completeTask(task.id);
			task.status = 'completed';
		} catch (e: any) {
			error = e.message || 'Abschluss fehlgeschlagen';
		} finally {
			completing = false;
		}
	}

	function formatDate(date: string): string {
		return new Date(date).toLocaleDateString('de-AT', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
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
</script>

<svelte:head>
	<title>{task?.title || 'Aufgabe'} | Mandantenportal</title>
</svelte:head>

<div class="space-y-6">
	<button
		class="inline-flex items-center gap-2 text-gray-600 hover:text-gray-900"
		on:click={() => goto('/tasks')}
	>
		<ArrowLeft class="w-4 h-4" />
		Zurück zu Aufgaben
	</button>

	{#if loading}
		<Card>
			<div class="h-64 bg-gray-100 rounded animate-pulse"></div>
		</Card>
	{:else if error && !task}
		<Card class="text-center py-12">
			<p class="text-red-600">{error}</p>
		</Card>
	{:else if task}
		{@const priority = getPriorityBadge(task.priority)}
		{@const overdue = task.due_date && task.status === 'open' && isOverdue(task.due_date)}

		<Card>
			<div class="flex items-start justify-between mb-6">
				<div class="flex items-center gap-4">
					<div class="p-3 rounded-lg" class:bg-green-100={task.status === 'completed'} class:bg-gray-100={task.status !== 'completed'}>
						{#if task.status === 'completed'}
							<CheckCircle class="w-8 h-8 text-green-600" />
						{:else}
							<ListTodo class="w-8 h-8 text-gray-600" />
						{/if}
					</div>
					<div>
						<h1 class="text-xl font-bold text-gray-900">
							{task.title}
						</h1>
						<p class="text-sm text-gray-500">
							Erstellt am {formatDate(task.created_at)}
						</p>
					</div>
				</div>

				<div class="flex gap-2">
					<Badge variant={priority.variant}>{priority.label}</Badge>
					{#if task.status === 'completed'}
						<Badge variant="success">Erledigt</Badge>
					{:else if overdue}
						<Badge variant="danger">Überfällig</Badge>
					{:else}
						<Badge variant="warning">Offen</Badge>
					{/if}
				</div>
			</div>

			{#if task.description}
				<div class="mb-6">
					<h2 class="text-sm font-medium text-gray-700 mb-2">Beschreibung</h2>
					<p class="text-gray-600 whitespace-pre-wrap">{task.description}</p>
				</div>
			{/if}

			{#if task.due_date}
				<div class="mb-6 p-4 rounded-lg" class:bg-red-50={overdue} class:border-red-200={overdue} class:bg-gray-50={!overdue} class:border-gray-200={!overdue}>
					<p class="text-sm font-medium" class:text-red-800={overdue} class:text-gray-700={!overdue}>
						Fälligkeitsdatum
					</p>
					<p class="text-lg font-semibold" class:text-red-900={overdue} class:text-gray-900={!overdue}>
						{formatDate(task.due_date)}
					</p>
				</div>
			{/if}

			{#if error}
				<div class="mb-6 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
					{error}
				</div>
			{/if}

			<!-- Linked resources -->
			{#if task.document_id || task.upload_id || task.approval_id}
				<div class="mb-6">
					<h2 class="text-sm font-medium text-gray-700 mb-3">Verknüpfte Ressourcen</h2>
					<div class="space-y-2">
						{#if task.document_id}
							<a
								href="/documents/{task.document_id}"
								class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
							>
								<FileText class="w-5 h-5 text-gray-400" />
								<span class="text-sm text-gray-700">Verknüpftes Dokument anzeigen</span>
							</a>
						{/if}
						{#if task.approval_id}
							<a
								href="/approvals/{task.approval_id}"
								class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
							>
								<CheckCircle class="w-5 h-5 text-gray-400" />
								<span class="text-sm text-gray-700">Verknüpfte Freigabe anzeigen</span>
							</a>
						{/if}
					</div>
				</div>
			{/if}

			{#if task.status === 'open'}
				<div class="flex gap-4">
					{#if task.upload_id === null && !task.document_id}
						<Button
							variant="outline"
							class="flex-1"
							on:click={() => goto('/upload')}
						>
							<Upload class="w-4 h-4 mr-2" />
							Beleg hochladen
						</Button>
					{/if}
					<Button
						class="flex-1"
						loading={completing}
						on:click={handleComplete}
					>
						<CheckCircle class="w-4 h-4 mr-2" />
						Als erledigt markieren
					</Button>
				</div>
			{:else if task.completed_at}
				<div class="p-4 bg-green-50 border border-green-200 rounded-lg text-center">
					<CheckCircle class="w-8 h-8 mx-auto mb-2 text-green-600" />
					<p class="text-green-800 font-medium">Aufgabe erledigt</p>
					<p class="text-sm text-green-600">{formatDate(task.completed_at)}</p>
				</div>
			{/if}
		</Card>
	{/if}
</div>
