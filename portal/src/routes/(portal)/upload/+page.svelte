<script lang="ts">
	import { Upload, X, FileText, CheckCircle } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { api } from '$lib/api/client';

	let files: File[] = [];
	let category = 'sonstiges';
	let note = '';
	let uploading = false;
	let uploadComplete = false;
	let error = '';
	let dragOver = false;

	const categories = [
		{ value: 'rechnung', label: 'Rechnung' },
		{ value: 'beleg', label: 'Beleg' },
		{ value: 'vertrag', label: 'Vertrag' },
		{ value: 'kontoauszug', label: 'Kontoauszug' },
		{ value: 'sonstiges', label: 'Sonstiges' }
	];

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;

		const droppedFiles = e.dataTransfer?.files;
		if (droppedFiles) {
			addFiles(droppedFiles);
		}
	}

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files) {
			addFiles(input.files);
		}
	}

	function addFiles(fileList: FileList) {
		const validTypes = ['application/pdf', 'image/jpeg', 'image/png',
			'application/msword', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
			'application/vnd.ms-excel', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'];

		const maxSize = 25 * 1024 * 1024; // 25MB

		for (const file of fileList) {
			if (!validTypes.includes(file.type)) {
				error = `${file.name}: Ungültiger Dateityp`;
				continue;
			}

			if (file.size > maxSize) {
				error = `${file.name}: Datei zu groß (max. 25MB)`;
				continue;
			}

			if (!files.some(f => f.name === file.name)) {
				files = [...files, file];
			}
		}
	}

	function removeFile(index: number) {
		files = files.filter((_, i) => i !== index);
	}

	async function handleUpload() {
		if (files.length === 0) {
			error = 'Bitte wählen Sie mindestens eine Datei aus';
			return;
		}

		uploading = true;
		error = '';

		try {
			for (const file of files) {
				await api.uploadFile(file, category, note || undefined);
			}

			uploadComplete = true;
			files = [];
			note = '';
		} catch (e: any) {
			error = e.message || 'Upload fehlgeschlagen';
		} finally {
			uploading = false;
		}
	}

	function resetForm() {
		uploadComplete = false;
		files = [];
		note = '';
		error = '';
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return bytes + ' B';
		if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
		return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
	}
</script>

<svelte:head>
	<title>Beleg hochladen | Mandantenportal</title>
</svelte:head>

<div class="max-w-2xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Beleg hochladen</h1>

	{#if uploadComplete}
		<Card class="text-center py-12">
			<CheckCircle class="w-16 h-16 mx-auto mb-4 text-green-500" />
			<h2 class="text-xl font-semibold text-gray-900 mb-2">Upload erfolgreich!</h2>
			<p class="text-gray-600 mb-6">
				Ihre Belege wurden hochgeladen und werden bearbeitet.
			</p>
			<Button on:click={resetForm}>Weitere Belege hochladen</Button>
		</Card>
	{:else}
		<Card>
			{#if error}
				<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm mb-6">
					{error}
				</div>
			{/if}

			<!-- Drop zone -->
			<div
				class="border-2 border-dashed rounded-lg p-8 text-center transition-colors"
				class:border-primary={dragOver}
				class:bg-primary/5={dragOver}
				class:border-gray-300={!dragOver}
				on:dragover|preventDefault={() => (dragOver = true)}
				on:dragleave={() => (dragOver = false)}
				on:drop={handleDrop}
				role="button"
				tabindex="0"
			>
				<Upload class="w-12 h-12 mx-auto mb-4 text-gray-400" />
				<p class="text-gray-600 mb-2">
					Dateien hierher ziehen oder
					<label class="text-primary hover:underline cursor-pointer">
						durchsuchen
						<input
							type="file"
							multiple
							accept=".pdf,.jpg,.jpeg,.png,.doc,.docx,.xls,.xlsx"
							class="hidden"
							on:change={handleFileSelect}
						/>
					</label>
				</p>
				<p class="text-xs text-gray-500">
					PDF, JPG, PNG, DOC, DOCX, XLS, XLSX (max. 25MB)
				</p>
			</div>

			<!-- File list -->
			{#if files.length > 0}
				<div class="mt-6 space-y-2">
					<p class="text-sm font-medium text-gray-700">Ausgewählte Dateien:</p>
					{#each files as file, index}
						<div class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg">
							<FileText class="w-5 h-5 text-gray-400" />
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium text-gray-900 truncate">{file.name}</p>
								<p class="text-xs text-gray-500">{formatFileSize(file.size)}</p>
							</div>
							<button
								class="p-1 text-gray-400 hover:text-red-500 transition-colors"
								on:click={() => removeFile(index)}
							>
								<X class="w-4 h-4" />
							</button>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Category selection -->
			<div class="mt-6">
				<label class="block text-sm font-medium text-gray-700 mb-2">
					Kategorie
				</label>
				<select
					bind:value={category}
					class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
				>
					{#each categories as cat}
						<option value={cat.value}>{cat.label}</option>
					{/each}
				</select>
			</div>

			<!-- Note -->
			<div class="mt-4">
				<label class="block text-sm font-medium text-gray-700 mb-2">
					Anmerkung (optional)
				</label>
				<textarea
					bind:value={note}
					rows="3"
					placeholder="Zusätzliche Informationen..."
					class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary resize-none"
				></textarea>
			</div>

			<!-- Submit -->
			<div class="mt-6">
				<Button
					class="w-full"
					loading={uploading}
					disabled={files.length === 0}
					on:click={handleUpload}
				>
					{files.length === 0 ? 'Dateien auswählen' : `${files.length} Datei${files.length !== 1 ? 'en' : ''} hochladen`}
				</Button>
			</div>
		</Card>
	{/if}
</div>
