<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import RideForm from '$lib/components/rides/RideForm.svelte';
	import { openNotifModal } from '$lib/notifModal';
	import { pushState } from '$lib/pwa';
	import { get } from 'svelte/store';
	import { m } from '$lib/paraglide/messages';

	let destinations = $state<string[]>([]);
	onMount(async () => { try { destinations = await api.destinations.list(); } catch { destinations = []; } });

	// Prefill from "I can drive this" on a requested ride (or any deep link).
	const sp = page.url.searchParams;
	const origin = sp.get('origin') ?? '';
	const destination = sp.get('destination') ?? '';
	const departureAt = (() => {
		const dep = sp.get('departure_at');
		if (!dep) return '';
		const d = new Date(dep);
		if (isNaN(d.getTime())) return '';
		const p = (n: number) => String(n).padStart(2, '0');
		return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
	})();

	function posted() {
		if (get(pushState) !== 'subscribed') openNotifModal(get(pushState));
		goto('/my-rides');
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.postRideTitle()}</h2>
<RideForm {destinations} {origin} {destination} {departureAt} onposted={posted} />
