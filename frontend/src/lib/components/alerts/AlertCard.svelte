<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { formatTime, formatDate, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Request } from '$lib/types';

	let { request, onseematches }: { request: Request; onseematches?: (o: string, d: string, dep: string) => void } = $props();
	let msg = $state('');
	let deleted = $state(false);

	const ZERO = '0001-01-01T00:00:00Z';
	const hasDate = $derived(request.Date !== ZERO && request.Date?.slice(0, 4) !== '0001');
	const hasTime = $derived(request.DepartureAt !== ZERO && request.DepartureAt?.slice(0, 4) !== '0001');
	const isDaily = $derived(hasTime && request.DepartureAt.slice(0, 10) === '1970-01-01');

	async function del() {
		try {
			await api.requests.del(request.ID, get(userPhone));
			msg = m.deleteOk();
			deleted = true;
		} catch {
			msg = m.deleteErr();
		}
	}
</script>

<div class="card rounded border p-3" id="card-{request.ID}" style:opacity={deleted ? 0.4 : 1}>
	<div class="card-route font-medium" translate="no">{request.Origin} <span class="route-arrow">→</span> {request.Destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		{#if !hasDate && !hasTime}
			<span class="tag tag-anytime">{m.alertAnytimeLabel()}</span>
		{:else if isDaily}
			<span class="tag tag-daily">{new Date(request.DepartureAt).toISOString().slice(11, 16)}</span>
			<span class="tag">{flexLabel(request.Flexibility)}</span>
		{:else if hasDate && !hasTime}
			<span class="tag">{formatDate(request.Date)}</span>
		{:else}
			<span>{formatTime(request.DepartureAt)}</span>
			<span class="tag">{flexLabel(request.Flexibility)}</span>
		{/if}
	</div>
	<div class="alert-actions flex gap-2">
		<button type="button" class="btn-see-matches" data-origin={request.Origin} data-dest={request.Destination}
			data-dept={hasTime ? request.DepartureAt : ''}
			onclick={() => onseematches?.(request.Origin, request.Destination, hasTime ? request.DepartureAt : '')}>{m.btnSeeMatches()}</button>
		<button type="button" class="btn btn-danger btn-delete" data-id={request.ID} data-phone={get(userPhone)} disabled={deleted} onclick={del}>{m.btnDelete()}</button>
	</div>
	<div class="delete-msg" id="msg-{request.ID}">{msg}</div>
</div>
