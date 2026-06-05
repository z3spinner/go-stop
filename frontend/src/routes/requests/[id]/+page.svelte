<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { formatTime, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Request } from '$lib/types';

	let req = $state<Request | null>(null);
	let err = $state('');
	onMount(async () => {
		try { req = await api.requests.get(page.params.id!, get(userPhone)); }
		catch (e) { err = e instanceof Error ? e.message : String(e); }
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.detailReqTitle()}</h2>
{#if err}
	<p class="error text-red-600">{err}</p>
{:else if req}
	<div class="card detail-card rounded border p-3">
		<div class="card-route font-medium" translate="no">{req.Origin} <span class="route-arrow">→</span> {req.Destination}</div>
		<div class="card-meta flex gap-2 text-sm text-gray-600"><span>{formatTime(req.DepartureAt)}</span><span class="tag">{flexLabel(req.Flexibility)}</span></div>
		<div class="detail-table mt-2">
			<div>{m.labelSearcher()}: {req.SearcherName}</div>
			<div>{m.labelContact()}: <a href="tel:{req.Phone}">{req.Phone}</a></div>
		</div>
	</div>
{:else}
	<p>…</p>
{/if}
