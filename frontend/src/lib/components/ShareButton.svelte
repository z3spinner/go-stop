<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '$lib/paraglide/messages';

	// `url` defaults to the current page. `title`/`text` are used by the native
	// share sheet; the clipboard fallback shares the URL. `size` sizes the icon.
	let {
		url,
		title,
		text,
		size = 18
	}: { url?: string; title: string; text?: string; size?: number } = $props();

	let copied = $state(false);
	let timer: ReturnType<typeof setTimeout> | undefined;

	async function share() {
		const shareUrl = url ?? (typeof window !== 'undefined' ? window.location.href : '');
		const nav = typeof navigator !== 'undefined' ? navigator : undefined;
		if (!shareUrl || !nav) return;

		// Native share sheet where supported (mobile). Accessed via a cast because
		// the Web Share API is optional at runtime even though lib.dom declares it.
		// A cancelled share rejects — that's not an error, so we just stop.
		const shareFn = (nav as { share?: (data: ShareData) => Promise<void> }).share;
		if (shareFn) {
			try {
				await shareFn.call(nav, { title, text, url: shareUrl });
			} catch {
				/* user dismissed the share sheet */
			}
			return;
		}

		// Desktop fallback: copy the link and confirm with a brief check icon.
		try {
			await nav.clipboard.writeText(shareUrl);
			copied = true;
			clearTimeout(timer);
			timer = setTimeout(() => (copied = false), 2000);
		} catch {
			/* clipboard unavailable (e.g. insecure context) — nothing we can do */
		}
	}
</script>

<button
	type="button"
	class="btn-share inline-flex items-center align-middle text-gray-500 hover:text-gray-800"
	aria-label={m.btnShare()}
	title={copied ? m.shareCopied() : m.btnShare()}
	onclick={share}
>
	{#if copied}
		<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12" /></svg>
	{:else}
		<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="18" cy="5" r="3" /><circle cx="6" cy="12" r="3" /><circle cx="18" cy="19" r="3" /><line x1="8.59" y1="13.51" x2="15.42" y2="17.49" /><line x1="15.41" y1="6.51" x2="8.59" y2="10.49" /></svg>
	{/if}
</button>
