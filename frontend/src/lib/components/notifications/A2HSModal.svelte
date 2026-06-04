<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { a2hsModalOpen } from '$lib/a2hs';
	import { m } from '$lib/paraglide/messages';
	let open = $derived($a2hsModalOpen);
	function onOpenChange(v: boolean) {
		a2hsModalOpen.set(v);
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-sm">
		<Dialog.Header>
			<Dialog.Title>{m.a2hsTitle()}</Dialog.Title>
		</Dialog.Header>
		<p class="text-sm text-gray-600">{m.a2hsBody()}</p>
		<ol class="a2hs-steps list-none space-y-1 text-sm text-gray-600">
			<li class="a2hs-step">{m.a2hsStep1()}</li>
			<li class="a2hs-step">{m.a2hsStep2()}</li>
			<li class="a2hs-step">{m.a2hsStep3()}</li>
		</ol>
		<p class="a2hs-note text-sm text-gray-500">{m.a2hsNote()}</p>
	</Dialog.Content>
</Dialog.Root>
