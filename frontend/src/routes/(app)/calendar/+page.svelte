<script lang="ts">
	import { formatDate } from '$lib/utils';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	interface Deadline {
		id: string;
		title: string;
		account: string;
		date: Date;
		type: 'uva' | 'zm' | 'ersuchen' | 'other';
	}

	// Current date info
	const now = new Date();
	let currentMonth = $state(now.getMonth());
	let currentYear = $state(now.getFullYear());

	const monthNames = [
		'Jänner', 'Februar', 'März', 'April', 'Mai', 'Juni',
		'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'
	];

	const dayNames = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'];

	// Mock deadlines
	const deadlines: Deadline[] = [
		{ id: '1', title: 'UVA Jänner', account: 'Muster GmbH', date: new Date(2025, 1, 15), type: 'uva' },
		{ id: '2', title: 'UVA Jänner', account: 'Test AG', date: new Date(2025, 1, 15), type: 'uva' },
		{ id: '3', title: 'ZM Q4/2024', account: 'Muster GmbH', date: new Date(2025, 1, 28), type: 'zm' },
		{ id: '4', title: 'Ergänzungsersuchen', account: 'Test AG', date: new Date(2025, 0, 20), type: 'ersuchen' },
		{ id: '5', title: 'UVA Dezember', account: 'Demo GmbH', date: new Date(2025, 0, 15), type: 'uva' },
	];

	// Generate calendar days
	let calendarDays = $derived(() => {
		const firstDay = new Date(currentYear, currentMonth, 1);
		const lastDay = new Date(currentYear, currentMonth + 1, 0);
		const daysInMonth = lastDay.getDate();

		// Get day of week (0 = Sunday, adjust for Monday start)
		let startDay = firstDay.getDay() - 1;
		if (startDay < 0) startDay = 6;

		const days: { date: Date | null; deadlines: Deadline[] }[] = [];

		// Add empty slots for days before the first
		for (let i = 0; i < startDay; i++) {
			days.push({ date: null, deadlines: [] });
		}

		// Add days of the month
		for (let day = 1; day <= daysInMonth; day++) {
			const date = new Date(currentYear, currentMonth, day);
			const dayDeadlines = deadlines.filter(d =>
				d.date.getFullYear() === currentYear &&
				d.date.getMonth() === currentMonth &&
				d.date.getDate() === day
			);
			days.push({ date, deadlines: dayDeadlines });
		}

		return days;
	});

	// Selected day
	let selectedDate = $state<Date | null>(null);
	let selectedDeadlines = $derived(
		selectedDate
			? deadlines.filter(d =>
				d.date.getFullYear() === selectedDate.getFullYear() &&
				d.date.getMonth() === selectedDate.getMonth() &&
				d.date.getDate() === selectedDate.getDate()
			)
			: []
	);

	// Upcoming deadlines this month
	let upcomingDeadlines = $derived(
		deadlines
			.filter(d => d.date >= now && d.date.getMonth() === currentMonth && d.date.getFullYear() === currentYear)
			.sort((a, b) => a.date.getTime() - b.date.getTime())
			.slice(0, 5)
	);

	function prevMonth() {
		if (currentMonth === 0) {
			currentMonth = 11;
			currentYear--;
		} else {
			currentMonth--;
		}
	}

	function nextMonth() {
		if (currentMonth === 11) {
			currentMonth = 0;
			currentYear++;
		} else {
			currentMonth++;
		}
	}

	function goToToday() {
		currentMonth = now.getMonth();
		currentYear = now.getFullYear();
	}

	function selectDay(date: Date | null) {
		selectedDate = date;
	}

	function isToday(date: Date): boolean {
		return date.toDateString() === now.toDateString();
	}

	function getTypeColor(type: Deadline['type']): string {
		switch (type) {
			case 'uva': return 'bg-[var(--color-accent)]';
			case 'zm': return 'bg-[var(--color-info)]';
			case 'ersuchen': return 'bg-[var(--color-warning)]';
			default: return 'bg-[var(--color-ink-muted)]';
		}
	}

	function getTypeLabel(type: Deadline['type']): string {
		switch (type) {
			case 'uva': return 'UVA';
			case 'zm': return 'ZM';
			case 'ersuchen': return 'Ersuchen';
			default: return 'Other';
		}
	}
</script>

<svelte:head>
	<title>Calendar - Austrian Business Infrastructure</title>
</svelte:head>

<div class="max-w-7xl mx-auto space-y-6 animate-in">
	<!-- Header -->
	<div>
		<h1 class="text-xl font-semibold text-[var(--color-ink)]">Calendar</h1>
		<p class="text-sm text-[var(--color-ink-muted)]">
			View and manage your filing deadlines
		</p>
	</div>

	<div class="grid lg:grid-cols-3 gap-6">
		<!-- Calendar -->
		<Card padding="none" class="lg:col-span-2">
			<!-- Month navigation -->
			<div class="p-4 border-b border-black/6 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<h2 class="text-lg font-semibold text-[var(--color-ink)]">
						{monthNames[currentMonth]} {currentYear}
					</h2>
					<Button variant="ghost" size="sm" onclick={goToToday}>Today</Button>
				</div>
				<div class="flex items-center gap-1">
					<button
						onclick={prevMonth}
						aria-label="Previous month"
						class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors"
					>
						<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="m15 18-6-6 6-6"/>
						</svg>
					</button>
					<button
						onclick={nextMonth}
						aria-label="Next month"
						class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-[var(--color-paper-inset)] transition-colors"
					>
						<svg class="w-5 h-5 text-[var(--color-ink-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="m9 18 6-6-6-6"/>
						</svg>
					</button>
				</div>
			</div>

			<!-- Day headers -->
			<div class="grid grid-cols-7 border-b border-black/6">
				{#each dayNames as day}
					<div class="py-2 text-center text-xs font-medium text-[var(--color-ink-muted)]">{day}</div>
				{/each}
			</div>

			<!-- Calendar grid -->
			<div class="grid grid-cols-7">
				{#each calendarDays() as day}
					<button
						onclick={() => selectDay(day.date)}
						disabled={!day.date}
						class={`
							min-h-[80px] p-2 border-b border-r border-black/4 text-left transition-colors
							${!day.date ? 'bg-[var(--color-paper-inset)] cursor-default' : 'hover:bg-[var(--color-paper-inset)]'}
							${day.date && selectedDate?.toDateString() === day.date.toDateString() ? 'bg-[var(--color-accent-muted)]' : ''}
						`}
					>
						{#if day.date}
							<span class={`
								inline-flex items-center justify-center w-7 h-7 text-sm rounded-full
								${isToday(day.date) ? 'bg-[var(--color-accent)] text-white font-medium' : 'text-[var(--color-ink)]'}
							`}>
								{day.date.getDate()}
							</span>

							{#if day.deadlines.length > 0}
								<div class="mt-1 space-y-1">
									{#each day.deadlines.slice(0, 2) as deadline}
										<div class="flex items-center gap-1">
											<div class={`w-2 h-2 rounded-full ${getTypeColor(deadline.type)}`}></div>
											<span class="text-xs text-[var(--color-ink-secondary)] truncate">{deadline.title}</span>
										</div>
									{/each}
									{#if day.deadlines.length > 2}
										<span class="text-xs text-[var(--color-ink-muted)]">+{day.deadlines.length - 2} more</span>
									{/if}
								</div>
							{/if}
						{/if}
					</button>
				{/each}
			</div>
		</Card>

		<!-- Sidebar -->
		<div class="space-y-6">
			<!-- Selected day details -->
			{#if selectedDate && selectedDeadlines.length > 0}
				<Card>
					<h3 class="font-semibold text-[var(--color-ink)] mb-3">
						{formatDate(selectedDate)}
					</h3>
					<div class="space-y-3">
						{#each selectedDeadlines as deadline}
							<div class="flex items-start gap-3 p-3 rounded-lg bg-[var(--color-paper-inset)]">
								<div class={`w-3 h-3 rounded-full mt-1 ${getTypeColor(deadline.type)}`}></div>
								<div class="flex-1 min-w-0">
									<p class="font-medium text-[var(--color-ink)]">{deadline.title}</p>
									<p class="text-sm text-[var(--color-ink-muted)]">{deadline.account}</p>
								</div>
								<Badge variant="default" size="sm">{getTypeLabel(deadline.type)}</Badge>
							</div>
						{/each}
					</div>
				</Card>
			{/if}

			<!-- Upcoming deadlines -->
			<Card>
				<h3 class="font-semibold text-[var(--color-ink)] mb-3">Upcoming</h3>
				{#if upcomingDeadlines.length === 0}
					<p class="text-sm text-[var(--color-ink-muted)]">No deadlines this month</p>
				{:else}
					<div class="space-y-3">
						{#each upcomingDeadlines as deadline}
							{@const daysUntil = Math.ceil((deadline.date.getTime() - now.getTime()) / 86400000)}
							<div class="flex items-center gap-3">
								<div class={`w-2 h-2 rounded-full ${getTypeColor(deadline.type)}`}></div>
								<div class="flex-1 min-w-0">
									<p class="text-sm font-medium text-[var(--color-ink)] truncate">{deadline.title}</p>
									<p class="text-xs text-[var(--color-ink-muted)]">{deadline.account}</p>
								</div>
								<Badge
									variant={daysUntil <= 3 ? 'error' : daysUntil <= 7 ? 'warning' : 'default'}
									size="sm"
								>
									{daysUntil}d
								</Badge>
							</div>
						{/each}
					</div>
				{/if}
			</Card>

			<!-- Legend -->
			<Card>
				<h3 class="font-semibold text-[var(--color-ink)] mb-3">Legend</h3>
				<div class="space-y-2">
					<div class="flex items-center gap-2">
						<div class="w-3 h-3 rounded-full bg-[var(--color-accent)]"></div>
						<span class="text-sm text-[var(--color-ink-secondary)]">UVA - VAT Return</span>
					</div>
					<div class="flex items-center gap-2">
						<div class="w-3 h-3 rounded-full bg-[var(--color-info)]"></div>
						<span class="text-sm text-[var(--color-ink-secondary)]">ZM - Zusammenfassende Meldung</span>
					</div>
					<div class="flex items-center gap-2">
						<div class="w-3 h-3 rounded-full bg-[var(--color-warning)]"></div>
						<span class="text-sm text-[var(--color-ink-secondary)]">Ersuchen - Official Request</span>
					</div>
				</div>
			</Card>
		</div>
	</div>
</div>
