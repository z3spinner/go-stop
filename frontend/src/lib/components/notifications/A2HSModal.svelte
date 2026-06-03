<script lang="ts">
	import { a2hsModalOpen } from '$lib/a2hs';
	import { m } from '$lib/paraglide/messages';
	function close() { a2hsModalOpen.set(false); }
</script>

<svelte:window onkeydown={(e) => { if ($a2hsModalOpen && e.key === 'Escape') close(); }} />

{#if $a2hsModalOpen}
	<div class="modal-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onclick={close} role="presentation">
		<div class="modal w-full max-w-sm rounded bg-white p-5" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="dialog" aria-modal="true" tabindex={-1}>
			<h3 class="mb-2 text-lg font-semibold">{m.a2hsTitle()}</h3>
			<p>{m.a2hsBody()}</p>
			<ol class="a2hs-steps mt-3 list-none space-y-1">
				<li class="a2hs-step">{m.a2hsStep1()}</li>
				<li class="a2hs-step">{m.a2hsStep2()}</li>
				<li class="a2hs-step">{m.a2hsStep3()}</li>
			</ol>
			<p class="a2hs-note mt-3 text-sm text-gray-500">{m.a2hsNote()}</p>
			<button type="button" class="mt-3 rounded border px-3 py-1" onclick={close}>{m.privacyClose()}</button>
		</div>
	</div>
{/if}
