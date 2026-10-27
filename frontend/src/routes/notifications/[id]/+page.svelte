<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { userPhone } from '#lib/stores.js';
	import { inboxAction, inboxIcon, inboxItems, inboxRoute, inboxTitle, refreshInbox } from '#lib/inbox.js';
	import { formatTime } from '#lib/utils.js';
	import { m } from '#lib/paraglide/messages.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Button } from '#lib/components/ui/button/index.js';

	// One notification on its own: the ones that point at no particular page
	// (morning digest, weekly reminder, test) open here from the list, so the
	// reader stays with what they tapped. The list already marked it read.
	const id = $derived(page.params.id);
	let phone = $derived($userPhone.trim());
	let item = $derived($inboxItems.find((i) => i.id === id));
	let action = $derived(item ? inboxAction(item) : null);
	let loadedFor = $state<string | null>(null);

	// Opened by a reload or a shared link, the inbox may not be loaded yet (or
	// loaded before this item arrived), so fetch past the throttle if it's missing.
	$effect(() => {
		const p = phone;
		if (!p) return;
		refreshInbox(p, { force: !untrack(() => item) }).then(() => {
			if (untrack(() => phone) === p) loadedFor = p;
		});
	});
</script>

{#if item}
	<Card.Root class="mx-auto max-w-md" data-kind={item.kind}>
		<Card.Header>
			<Card.Title class="flex items-start gap-2 text-lg">
				<span aria-hidden="true">{inboxIcon(item)}</span>
				<span>{inboxTitle(item)}</span>
			</Card.Title>
			<Card.Description class="flex flex-wrap items-center gap-2">
				<span>{formatTime(item.created_at)}</span>
				{#if item.kind === 'test'}<span class="tag">{m.inboxTestTag()}</span>{/if}
			</Card.Description>
		</Card.Header>
		<Card.Content>
			<p class="notif-body whitespace-pre-line" translate="no">{inboxRoute(item)}</p>
		</Card.Content>
		{#if action}
			<Card.Footer>
				<Button size="lg" class="notif-action w-full" href={action.href}>{action.label}</Button>
			</Card.Footer>
		{/if}
	</Card.Root>
{:else if !phone || loadedFor === phone}
	<p class="notif-missing text-gray-600">{m.inboxNotFound()}</p>
{/if}
