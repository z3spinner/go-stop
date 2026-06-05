<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { goto } from '$app/navigation';
	import { pollToasts } from '$lib/pwa';
	import { m } from '$lib/paraglide/messages';
	import type { NotificationItem } from '$lib/types';

	function dismiss(n: NotificationItem) { pollToasts.update((t) => t.filter((x) => x !== n)); }
	function view(n: NotificationItem) { dismiss(n); goto('/my-searches'); }
</script>

<div class="poll-toast-host fixed bottom-3 left-1/2 z-50 flex -translate-x-1/2 flex-col gap-2">
	{#each $pollToasts as n}
		<div class="poll-toast flex items-center gap-2 rounded bg-gray-900 p-3 text-white shadow">
			<span class="poll-toast-body">🚗 {n.driver_name} · {n.origin} → {n.destination}</span>
			<button type="button" class="poll-toast-view underline" onclick={() => view(n)}>{m.pollToastView()}</button>
			<button type="button" class="poll-toast-close" aria-label="Dismiss" onclick={() => dismiss(n)}>✕</button>
		</div>
	{/each}
</div>
