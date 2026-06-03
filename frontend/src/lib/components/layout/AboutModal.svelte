<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { config } from '$lib/config';
	let { onclose }: { onclose: () => void } = $props();
</script>

<div class="modal-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onclick={onclose} role="presentation">
	<div class="modal max-h-[85vh] w-full max-w-lg overflow-auto rounded bg-white p-5" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
		<h3 class="mb-2 text-lg font-semibold">{m.aboutTitle()}</h3>
		<div class="prose prose-sm">{@html m.aboutBody({ siteName: $config.siteName })}</div>
		<button type="button" class="btn-modal-close mt-4 rounded border px-3 py-1" onclick={onclose}>{m.privacyClose()}</button>
	</div>
</div>
