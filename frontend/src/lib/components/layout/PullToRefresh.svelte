<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!--
  Custom pull-to-refresh for the installed (standalone) iOS PWA, where iOS
  provides no native pull-to-refresh. In a normal browser tab we do nothing so
  the browser's own gesture keeps working. The actual refresh is delegated to the
  `onrefresh` callback (the layout re-fetches page data).
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { isStandalone } from '$lib/pwa';

	let { onrefresh }: { onrefresh: () => Promise<void> | void } = $props();

	const THRESHOLD = 70; // damped px the user must pull to trigger a refresh
	const MAX = 110; // cap on indicator travel
	const DAMP = 0.5; // pull resistance

	let pull = $state(0); // damped distance currently shown
	let refreshing = $state(false);

	let armed = false; // gesture began at the top with a single finger
	let decided = false; // vertical-vs-horizontal direction locked in
	let startY = 0;
	let startX = 0;

	const progress = $derived(Math.min(1, pull / THRESHOLD));
	const visible = $derived(pull > 0 || refreshing);

	onMount(() => {
		// Browser tabs keep their native pull-to-refresh; only the chrome-less
		// standalone PWA needs this.
		if (!isStandalone()) return;

		const onStart = (e: TouchEvent) => {
			if (refreshing || e.touches.length !== 1 || window.scrollY > 0) {
				armed = false;
				return;
			}
			armed = true;
			decided = false;
			startY = e.touches[0].clientY;
			startX = e.touches[0].clientX;
		};

		const onMove = (e: TouchEvent) => {
			if (!armed || refreshing) return;
			const dy = e.touches[0].clientY - startY;
			const dx = e.touches[0].clientX - startX;
			if (!decided) {
				// Let horizontal swipes (e.g. the home feed's horizontal scroller) pass through.
				if (Math.abs(dx) > Math.abs(dy)) {
					armed = false;
					return;
				}
				if (dy <= 0) return; // wait for downward intent
				decided = true;
			}
			if (dy <= 0) {
				pull = 0;
				return;
			}
			pull = Math.min(MAX, dy * DAMP);
			// Own the gesture: stop the body's rubber-band overscroll.
			e.preventDefault();
		};

		const onEnd = async () => {
			if (!armed || refreshing) {
				armed = false;
				return;
			}
			armed = false;
			if (pull < THRESHOLD) {
				pull = 0;
				return;
			}
			refreshing = true;
			pull = THRESHOLD;
			const started = performance.now();
			try {
				await onrefresh();
			} finally {
				// Keep the spinner up briefly so a fast refresh still reads as one.
				const elapsed = performance.now() - started;
				if (elapsed < 500) await new Promise((r) => setTimeout(r, 500 - elapsed));
				refreshing = false;
				pull = 0;
			}
		};

		// touchmove must be non-passive so preventDefault() can suppress overscroll.
		window.addEventListener('touchstart', onStart, { passive: true });
		window.addEventListener('touchmove', onMove, { passive: false });
		window.addEventListener('touchend', onEnd, { passive: true });
		window.addEventListener('touchcancel', onEnd, { passive: true });
		return () => {
			window.removeEventListener('touchstart', onStart);
			window.removeEventListener('touchmove', onMove);
			window.removeEventListener('touchend', onEnd);
			window.removeEventListener('touchcancel', onEnd);
		};
	});
</script>

{#if visible}
	<div
		class="ptr-host"
		aria-hidden="true"
		style="transform: translate(-50%, {pull}px); transition: {refreshing || pull === 0
			? 'transform 0.2s ease'
			: 'none'};"
	>
		<div class="ptr-badge" style="opacity: {Math.min(1, progress + 0.25)}; scale: {0.7 + 0.3 * progress};">
			<span
				class="ptr-spinner"
				class:spin={refreshing}
				style={refreshing ? '' : `transform: rotate(${progress * 300}deg)`}
			></span>
		</div>
	</div>
{/if}

<style>
	.ptr-host {
		position: fixed;
		top: calc(env(safe-area-inset-top, 0px) - 30px);
		left: 50%;
		z-index: 40;
		pointer-events: none;
	}
	.ptr-badge {
		display: grid;
		place-items: center;
		width: 34px;
		height: 34px;
		border-radius: 9999px;
		background: white;
		box-shadow: 0 2px 10px rgba(0, 0, 0, 0.18);
	}
	.ptr-spinner {
		display: block;
		width: 18px;
		height: 18px;
		border: 2px solid var(--gray-300, #d1d5db);
		border-top-color: #28a836; /* brand primary */
		border-radius: 9999px;
	}
	.ptr-spinner.spin {
		animation: ptr-rot 0.7s linear infinite;
	}
	@keyframes ptr-rot {
		to {
			transform: rotate(360deg);
		}
	}
</style>
