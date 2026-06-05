<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import AlertForm from '$lib/components/alerts/AlertForm.svelte';
	import { openNotifModal } from '$lib/notifModal';
	import { pushState } from '$lib/pwa';
	import { get } from 'svelte/store';
	import { m } from '$lib/paraglide/messages';

	let destinations = $state<string[]>([]);
	const origin = $derived(page.url.searchParams.get('origin') ?? '');
	const destination = $derived(page.url.searchParams.get('destination') ?? '');
	const departureAt = $derived(page.url.searchParams.get('departure_at') ?? '');

	onMount(async () => { try { destinations = await api.destinations.list(); } catch { destinations = []; } });

	function posted() {
		if (get(pushState) !== 'subscribed') openNotifModal(get(pushState));
		goto('/');
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.notifRouteTitle()}</h2>
<p class="section-hint mb-3 text-sm text-gray-600">{m.notifRouteBody()}</p>
<AlertForm {origin} {destination} {departureAt} {destinations} onposted={posted} />
