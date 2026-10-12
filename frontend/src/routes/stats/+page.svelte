<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { m } from '$lib/paraglide/messages';
	import type { Stats, ActivityCounts } from '$lib/types';

	let stats = $state<Stats | null>(null);
	let err = $state(false);
	onMount(async () => { try { stats = await api.stats.get(); } catch { err = true; } });

	const tables = $derived(stats ? [
		{ title: m.statsSearches(), c: stats.searches },
		{ title: m.statsRidesPosted(), c: stats.rides_posted },
		{ title: m.statsConnections(), c: stats.connections },
		{ title: m.statsUnanswered(), c: stats.unanswered }
	] : []);
	function rows(c: ActivityCounts) {
		return [
			{ label: m.statsThisMonth(), n: c.this_month },
			{ label: m.statsThisYear(), n: c.this_year },
			{ label: m.statsAllTime2(), n: c.all_time }
		];
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.statsPageTitle()}</h2>
<div id="stats-content">
	{#if err}
		<p class="error text-red-600">⚠</p>
	{:else if stats}
		{#if stats.total_confirmed > 0}<div class="stats-total font-semibold">{m.statsAllTime({ n: stats.total_confirmed })}</div>{/if}
		<div class="stats-week-title mt-2 font-medium">{m.statsTitle()}</div>
		{#if stats.top_routes.length > 0}
			{#each stats.top_routes as rt}
				<div class="stats-row flex justify-between"><span translate="no">{rt.Origin} → {rt.Destination}</span><span class="stats-count">{m.statsRouteCount({ n: rt.Count })}</span></div>
			{/each}
		{:else}
			<p class="section-hint text-gray-500">{m.statsEmpty()}</p>
		{/if}
		<div class="activity-stats mt-4 grid grid-cols-2 gap-4">
			{#each tables as t}
				<div class="activity-stat">
					<div class="activity-stat-title font-medium">{t.title}</div>
					<div class="activity-stat-rows">
						{#each rows(t.c) as r}<div class="activity-row flex justify-between"><span>{r.label}</span><span>{r.n}</span></div>{/each}
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<p>…</p>
	{/if}
</div>
