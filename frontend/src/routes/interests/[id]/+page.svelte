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
	import * as Card from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import CheckIcon from '@lucide/svelte/icons/check';

	let contact = $state<ContactInfo | null>(null);
	let err = $state('');
	const id = $derived(page.params.id);

	onMount(async () => {
		try { contact = await api.interests.getContact(id!, get(userPhone)); }
		catch (e) { err = e instanceof Error ? e.message : String(e); }
	});
</script>

<div id="contact-result">
	{#if err}
		<p class="error text-red-600">{err}</p>
	{:else if contact}
		<Card.Root class="mx-auto max-w-sm">
			<Card.Header>
				<h2 class="flex items-center gap-1.5 text-sm font-semibold text-primary">
					<CheckIcon class="size-4" strokeWidth={2.5} />
					{m.contactRevealed()}
				</h2>
				<Card.Title class="text-lg" translate="no">{contact.origin} <span class="route-arrow">→</span> {contact.destination}</Card.Title>
				<Card.Description>{formatTime(contact.departure_at)}</Card.Description>
			</Card.Header>
			<Card.Content class="space-y-1.5 text-sm">
				<div>{contact.role === 'driver' ? m.labelDriver() : m.labelSearcher()}: <span class="font-medium">{contact.name}</span></div>
				<div>{m.theirNumber()} <a class="font-medium underline underline-offset-2" href="tel:{contact.phone}">{contact.phone}</a></div>
			</Card.Content>
			<Card.Footer class="flex-col items-stretch gap-2">
				<Button size="lg" class="w-full" href="tel:{contact.phone}">{m.btnCallNow()}</Button>
				<Button size="lg" variant="outline" class="w-full" id="btn-search-route" data-origin={contact.origin} data-dest={contact.destination}
					onclick={() => contact && goto(`/search?origin=${encodeURIComponent(contact.origin)}&destination=${encodeURIComponent(contact.destination)}`)}>{m.btnSearchRoute()}</Button>
			</Card.Footer>
		</Card.Root>
	{:else}
		<p>…</p>
	{/if}
</div>
