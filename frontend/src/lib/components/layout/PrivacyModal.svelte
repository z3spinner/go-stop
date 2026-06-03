<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	let { onclose }: { onclose: () => void } = $props();
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') onclose(); }} />
<div class="modal-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onclick={onclose} role="presentation">
	<div class="modal max-h-[85vh] w-full max-w-lg overflow-auto rounded bg-white p-5" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="dialog" aria-modal="true" tabindex={-1}>
		<h3 class="mb-2 text-lg font-semibold">{m.privacyTitle()}</h3>
		<div class="prose prose-sm">{@html m.privacyBody()}</div>
		<button type="button" class="btn-modal-close mt-4 rounded border px-3 py-1" onclick={onclose}>{m.privacyClose()}</button>
	</div>
</div>
