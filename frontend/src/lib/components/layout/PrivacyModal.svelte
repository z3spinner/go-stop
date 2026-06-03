<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	let { onclose }: { onclose: () => void } = $props();
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') onclose(); }} />
<div class="modal-overlay" onclick={onclose} role="presentation">
	<div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="dialog" aria-modal="true" tabindex={-1}>
		<div class="modal-header">
			<h2>{m.privacyTitle()}</h2>
			<button type="button" class="btn-modal-close" onclick={onclose}>{m.privacyClose()}</button>
		</div>
		<div class="modal-body">{@html m.privacyBody()}</div>
	</div>
</div>
