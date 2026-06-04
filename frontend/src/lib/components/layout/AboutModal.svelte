<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import { config } from '$lib/config';
	let { open = false, onclose }: { open?: boolean; onclose: () => void } = $props();
	function onOpenChange(v: boolean) {
		if (!v) onclose();
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-sm max-h-[85vh] overflow-y-auto">
		<Dialog.Header>
			<Dialog.Title>{m.aboutTitle()}</Dialog.Title>
		</Dialog.Header>
		<div class="modal-body text-sm text-gray-600">{@html m.aboutBody({ siteName: $config.siteName })}</div>
	</Dialog.Content>
</Dialog.Root>
