<script lang="ts">
	import LangPicker from './LangPicker.svelte';
	import BellButton from '$lib/components/notifications/BellButton.svelte';
	import { userName } from '$lib/stores';

	let { onabout, onprivacy }: { onabout?: () => void; onprivacy?: () => void } = $props();
	let hasProfile = $derived($userName.length > 0);
</script>

<div class="controls flex items-center gap-2">
	<LangPicker />
	<div class="controls-icons ml-auto flex items-center gap-2">
		<a id="btn-me" href="/me" class="me-icon" class:me-icon-set={hasProfile} aria-label="Profile" title="Me">👤</a>
		<button type="button" class="btn-about" aria-label="About" onclick={onabout}>ⓘ</button>
		<BellButton />
	</div>
</div>
