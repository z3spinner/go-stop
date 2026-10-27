<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { api } from '#lib/api.js';
	import { formatTime, flexLabel } from '#lib/utils.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { PublicRequest } from '#lib/types.js';

	let { request, rideId, driverPhone }: { request: PublicRequest; rideId: string; driverPhone: string } = $props();
	let done = $state(false);
	let busy = $state(false);

	async function ping() {
		if (busy || done) return;
		busy = true;
		try {
			await api.requests.ping(request.ID, rideId, driverPhone);
			done = true;
		} finally {
			busy = false;
		}
	}
</script>

<div class="seeker-row flex items-center gap-2">
	<div class="grow">
		<div>{request.SearcherName}</div>
		<div class="text-sm text-gray-600">{formatTime(request.DepartureAt)} · {flexLabel(request.Flexibility)}</div>
	</div>
	<button type="button" class="btn-ping-searcher" data-req-id={request.ID} data-ride-id={rideId} disabled={busy || done} onclick={ping}>
		{done ? '✓' : m.btnPingSearcher()}
	</button>
</div>
