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
	import * as Card from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import HandIcon from '@lucide/svelte/icons/hand';

	let req = $state<Request | null>(null);
	let err = $state('');
	onMount(async () => {
		try { req = await api.requests.get(page.params.id!, get(userPhone)); }
		catch (e) { err = e instanceof Error ? e.message : String(e); }
	});
</script>

{#if err}
	<p class="error text-red-600">{err}</p>
{:else if req}
	<Card.Root class="mx-auto max-w-sm">
		<Card.Header>
			<h2 class="flex items-center gap-1.5 text-sm font-semibold text-primary">
				<HandIcon class="size-4" strokeWidth={2.5} />
				{m.detailReqTitle()}
			</h2>
			<Card.Title class="text-lg" translate="no">{req.Origin} <span class="route-arrow">→</span> {req.Destination}</Card.Title>
			<Card.Description class="flex items-center gap-2">
				<span>{formatTime(req.DepartureAt)}</span>
				<Badge variant="secondary">{flexLabel(req.Flexibility)}</Badge>
			</Card.Description>
		</Card.Header>
		<Card.Content class="space-y-1.5 text-sm">
			<div>{m.labelSearcher()}: <span class="font-medium">{req.SearcherName}</span></div>
			<div>{m.labelContact()}: <a class="font-medium underline underline-offset-2" href="tel:{req.Phone}">{req.Phone}</a></div>
		</Card.Content>
		<Card.Footer class="flex-col items-stretch gap-2">
			<Button size="lg" class="w-full" href="tel:{req.Phone}">{m.btnCallNow()}</Button>
		</Card.Footer>
	</Card.Root>
{:else}
	<p>…</p>
{/if}
