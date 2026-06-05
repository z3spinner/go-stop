<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	let { open = false, onclose }: { open?: boolean; onclose: () => void } = $props();
	function onOpenChange(v: boolean) {
		if (!v) onclose();
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-md sm:max-w-md max-h-[85vh] overflow-y-auto">
		<Dialog.Header>
			<Dialog.Title>{m.privacyTitle()}</Dialog.Title>
		</Dialog.Header>
		<div class="modal-body">{@html m.privacyBody()}</div>
	</Dialog.Content>
</Dialog.Root>
