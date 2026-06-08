<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!--
  Native-feel pull-to-refresh for the installed (standalone) iOS PWA, where iOS
  provides no built-in pull-to-refresh. Wraps the page content: pulling drags the
  whole content down (1:1 with the finger), revealing an iOS-style "spoke"
  activity indicator in the gap above; past the threshold it refreshes via the
  `onrefresh` callback, then springs back. In a normal browser tab we do nothing,
  so the browser's own gesture keeps working.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';
	import { isStandalone } from '$lib/pwa';

	let { onrefresh, children }: { onrefresh: () => Promise<void> | void; children?: Snippet } =
		$props();

	const THRESHOLD = 70; // damped px the user must pull to trigger a refresh
	const MAX = 110; // cap on pull travel
	const DAMP = 0.5; // pull resistance
	const REST = 56; // content offset held while refreshing
	const SPOKES = 12;

	let pull = $state(0); // damped drag distance
	let refreshing = $state(false);
	let settling = $state(false); // animate the content back (transition on)

	let armed = false; // gesture began at the top with a single finger
	let decided = false; // vertical-vs-horizontal direction locked in
	let startY = 0;
	let startX = 0;

	// How far the content is translated down.
	const offset = $derived(refreshing ? REST : pull);
	const progress = $derived(Math.min(1, pull / THRESHOLD));
	// Spokes lit while pulling (fill clockwise); all lit (spinning) while refreshing.
	const litSpokes = $derived(refreshing ? SPOKES : Math.round(progress * SPOKES));
	const indicatorOpacity = $derived(refreshing ? 1 : Math.min(1, progress * 1.2));

	function spokeOpacity(i: number): number {
		if (refreshing) {
			// Fading gradient around the ring → a leading bright spoke "chases" as it spins.
			return 0.25 + (0.75 * i) / (SPOKES - 1);
		}
		return i < litSpokes ? 1 : 0.12; // reveal in pull order
	}

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
			settling = false;
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
				settling = true; // spring back
				pull = 0;
				return;
			}
			refreshing = true; // holds content at REST and spins the indicator
			const started = performance.now();
			try {
				await onrefresh();
			} finally {
				// Keep the spinner up briefly so a fast refresh still reads as one.
				const elapsed = performance.now() - started;
				if (elapsed < 500) await new Promise((r) => setTimeout(r, 500 - elapsed));
				settling = true;
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

<div
	class="ptr-track"
	style="transform: translateY({offset}px); transition: {settling || refreshing
		? 'transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1)'
		: 'none'};"
	ontransitionend={() => (settling = false)}
>
	{#if offset > 0 || refreshing}
		<div class="ptr-indicator" aria-hidden="true" style="opacity: {indicatorOpacity};">
			<div class="ptr-spokes" class:spin={refreshing}>
				{#each Array(SPOKES) as _, i (i)}
					<span
						class="ptr-spoke"
						style="transform: rotate({(i * 360) / SPOKES}deg) translateY(-6px); opacity: {spokeOpacity(i)};"
					></span>
				{/each}
			</div>
		</div>
	{/if}

	{@render children?.()}
</div>

<style>
	.ptr-track {
		position: relative; /* containing block for the absolutely-positioned indicator */
		will-change: transform;
	}
	/* Sits just above the content so it descends into the gap as the page pulls down. */
	.ptr-indicator {
		position: absolute;
		top: -34px;
		left: 50%;
		width: 28px;
		height: 28px;
		transform: translateX(-50%);
		pointer-events: none;
		z-index: 40;
	}
	.ptr-spokes {
		position: relative;
		width: 100%;
		height: 100%;
		color: #28a836; /* brand primary */
	}
	.ptr-spokes.spin {
		animation: ptr-spin 0.8s steps(12) infinite;
	}
	.ptr-spoke {
		position: absolute;
		top: 50%;
		left: 50%;
		width: 2px;
		height: 8px;
		margin: -4px 0 0 -1px; /* center the bar, then push out via translateY */
		border-radius: 1px;
		background: currentColor;
		transform-origin: center;
	}
	@keyframes ptr-spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.ptr-spokes.spin {
			animation-duration: 1.6s;
		}
	}
</style>
