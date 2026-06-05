<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { formatTime } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { ContactInfo } from '$lib/types';

	let contact = $state<ContactInfo | null>(null);
	let err = $state('');
	const id = $derived(page.params.id);

	onMount(async () => {
		try { contact = await api.interests.getContact(id!, get(userPhone)); }
		catch (e) { err = e instanceof Error ? e.message : String(e); }
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.contactRevealed()}</h2>
<div id="contact-result">
	{#if err}
		<p class="error text-red-600">{err}</p>
	{:else if contact}
		<div class="card contact-card rounded border p-3">
			<div class="card-route font-medium" translate="no">{contact.origin} <span class="route-arrow">→</span> {contact.destination}</div>
			<div class="card-meta text-sm text-gray-600">{formatTime(contact.departure_at)}</div>
			<div class="detail-table mt-2">
				<div>{contact.role === 'driver' ? m.labelDriver() : m.labelSearcher()}: {contact.name}</div>
				<div>{m.theirNumber()} <a href="tel:{contact.phone}">{contact.phone}</a></div>
			</div>
			<a class="btn btn-primary" href="tel:{contact.phone}">{m.btnCallNow()}</a>
			<button type="button" class="btn btn-secondary" id="btn-search-route" data-origin={contact.origin} data-dest={contact.destination}
				onclick={() => contact && goto(`/search?origin=${encodeURIComponent(contact.origin)}&destination=${encodeURIComponent(contact.destination)}`)}>{m.btnSearchRoute()}</button>
		</div>
	{:else}
		<p>…</p>
	{/if}
</div>
