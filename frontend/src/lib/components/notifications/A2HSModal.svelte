<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { a2hsModalOpen } from '$lib/a2hs';
	import { isInAppBrowser } from '$lib/pwa';
	import { m } from '$lib/paraglide/messages';
	let open = $derived($a2hsModalOpen);
	function onOpenChange(v: boolean) {
		a2hsModalOpen.set(v);
	}

	// Inside WhatsApp, Facebook, Instagram… the Share menu can't add to the Home
	// Screen, so the steps would lead nowhere: send them to Safari instead.
	let inApp = $derived(open && isInAppBrowser(navigator.userAgent));
	let link = $derived(open ? location.origin : '');
	let copied = $state(false);

	async function copy() {
		try {
			await navigator.clipboard.writeText(link);
			copied = true;
		} catch {
			/* no clipboard: the address is shown and selectable */
		}
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-sm">
		<Dialog.Header>
			<Dialog.Title>{m.a2hsTitle()}</Dialog.Title>
		</Dialog.Header>
		{#if inApp}
			<p class="a2hs-inapp text-sm text-gray-700">{m.a2hsInAppBody()}</p>
			<p class="text-sm text-gray-600">{m.a2hsInAppHow()}</p>
			<p class="a2hs-link select-all rounded bg-gray-100 px-2 py-1 text-center font-medium" translate="no">{link}</p>
			<Button class="a2hs-copy w-full" onclick={copy}>{m.a2hsCopyLink()}</Button>
			{#if copied}<p class="a2hs-copied text-center text-sm text-green-700" role="status">{m.a2hsCopied()}</p>{/if}
		{:else}
			<p class="text-sm text-gray-600">{m.a2hsBody()}</p>
			<ol class="a2hs-steps list-none">
				<li class="a2hs-step">{m.a2hsStep1()}</li>
				<li class="a2hs-step">{m.a2hsStep2()}</li>
				<li class="a2hs-step">{m.a2hsStep3()}</li>
			</ol>
		{/if}
		<p class="a2hs-note text-sm text-gray-500">{m.a2hsNote()}</p>
	</Dialog.Content>
</Dialog.Root>
