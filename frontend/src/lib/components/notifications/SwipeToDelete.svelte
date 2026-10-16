<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!--
  Swipe-to-delete row, like a phone's notification shade: drag the row left or
  right past ~40% of its width (or flick it) and it slides away, the row
  collapses and `ondelete` fires. A shorter drag springs back, a tap still opens
  the link inside, and vertical movement is left to the page scroll
  (touch-action: pan-y). Mouse and keyboard users get a delete button that
  shows on hover/focus; screen-reader users can reach it on touch devices too.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { swipeDirection, swipeOutcome } from '$lib/swipe';

	let {
		ondelete,
		deleteLabel,
		actionLabel,
		children
	}: {
		ondelete: () => void;
		/** Accessible name of the delete button, e.g. "Delete notification". */
		deleteLabel: string;
		/** Text shown on the red background while swiping, e.g. "Delete". */
		actionLabel: string;
		children: Snippet;
	} = $props();

	const SLIDE_MS = 180;
	const COLLAPSE_MS = 160;

	let row: HTMLDivElement;
	let dx = $state(0);
	let dragging = $state(false);
	let leaving = $state(false);
	let collapsing = $state(false);

	let pointerId: number | null = null;
	let startX = 0;
	let startY = 0;
	let startT = 0;
	let direction: 'h' | 'v' | null = null;
	let suppressClick = false;

	function reducedMotion(): boolean {
		return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
	}

	/** Slide out, collapse, then delete. Also used by the button / Delete key. */
	function dismiss(dir: -1 | 1 = -1) {
		if (leaving) return;
		leaving = true;
		dragging = false;
		const width = row?.offsetWidth || 400;
		dx = dir * width;
		const fast = reducedMotion();
		setTimeout(
			() => {
				collapsing = true;
				setTimeout(() => ondelete(), fast ? 0 : COLLAPSE_MS);
			},
			fast ? 0 : SLIDE_MS
		);
	}

	function onpointerdown(e: PointerEvent) {
		if (leaving || (e.pointerType === 'mouse' && e.button !== 0)) return;
		pointerId = e.pointerId;
		startX = e.clientX;
		startY = e.clientY;
		startT = performance.now();
		direction = null;
		suppressClick = false;
	}

	function onpointermove(e: PointerEvent) {
		if (pointerId === null || e.pointerId !== pointerId) return;
		const mx = e.clientX - startX;
		const my = e.clientY - startY;
		if (direction === null) {
			direction = swipeDirection(mx, my);
			if (direction === null) return;
			if (direction === 'h') {
				dragging = true;
				suppressClick = true; // this gesture is a swipe, not a tap
				row.setPointerCapture?.(e.pointerId);
			}
		}
		if (direction !== 'h') return;
		e.preventDefault();
		dx = mx;
	}

	function onpointerup(e: PointerEvent) {
		if (pointerId === null || e.pointerId !== pointerId) return;
		pointerId = null;
		if (direction !== 'h') {
			direction = null;
			return;
		}
		direction = null;
		dragging = false;
		const out = swipeOutcome(dx, row?.offsetWidth || 0, performance.now() - startT);
		if (out === 0) dx = 0;
		else dismiss(out);
	}

	function onpointercancel() {
		pointerId = null;
		direction = null;
		dragging = false;
		if (!leaving) dx = 0;
	}

	// A swipe that ends over the link must not also open it.
	function onclickcapture(e: MouseEvent) {
		if (suppressClick) {
			e.preventDefault();
			e.stopPropagation();
			suppressClick = false;
		}
	}

	function onkeydown(e: KeyboardEvent) {
		// Delete, or the Mac keyboard's delete key (Backspace).
		if ((e.key === 'Delete' || e.key === 'Backspace') && !leaving) {
			e.preventDefault();
			dismiss(-1);
		}
	}
</script>

<div class="swipe-row" class:swipe-collapsing={collapsing} style="--collapse-ms: {COLLAPSE_MS}ms">
	<div class="swipe-clip">
		<div
			class="swipe-behind"
			class:swipe-behind-left={dx < 0}
			style="opacity: {dx === 0 ? 0 : 1}"
			aria-hidden="true"
		>
			<span class="swipe-behind-label">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/></svg>
				{actionLabel}
			</span>
		</div>
		<!-- The gesture surface wraps a link (tap) and a real button (delete); the
		     swipe itself has keyboard/AT equivalents in that button and the
		     Delete key. -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			bind:this={row}
			class="swipe-front"
			class:swipe-dragging={dragging}
			class:swipe-leaving={leaving}
			style="transform: translateX({dx}px); --slide-ms: {SLIDE_MS}ms"
			{onpointerdown}
			{onpointermove}
			{onpointerup}
			{onpointercancel}
			onclickcapture={onclickcapture}
			{onkeydown}
		>
			{@render children()}
			<button type="button" class="swipe-delete-btn" aria-label={deleteLabel} title={deleteLabel} onclick={() => dismiss(-1)}>
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18"/><path d="M6 6l12 12"/></svg>
			</button>
		</div>
	</div>
</div>

<style>
	/* Collapses the row's height after it has slid away (grid 1fr → 0fr). */
	.swipe-row {
		display: grid;
		grid-template-rows: 1fr;
		transition: grid-template-rows var(--collapse-ms) ease;
	}
	.swipe-row.swipe-collapsing {
		grid-template-rows: 0fr;
	}
	.swipe-clip {
		position: relative;
		min-height: 0;
		overflow: hidden;
		border-radius: var(--radius, 8px);
	}
	.swipe-behind {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: flex-start;
		padding: 0 18px;
		background: #dc2626;
		color: #fff;
		font-weight: 600;
		font-size: 0.875rem;
	}
	.swipe-behind-left {
		justify-content: flex-end;
	}
	.swipe-behind-label {
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}
	.swipe-front {
		position: relative;
		touch-action: pan-y;
		transition: transform 0.2s ease;
	}
	/* The row is the card: no outer margin, or the red layer shows below it. */
	.swipe-front :global(.card) {
		margin-bottom: 0;
	}
	.swipe-front.swipe-dragging {
		transition: none;
		user-select: none;
		-webkit-user-select: none;
	}
	.swipe-front.swipe-leaving {
		transition: transform var(--slide-ms) ease-in;
	}
	.swipe-delete-btn {
		position: absolute;
		top: 6px;
		right: 6px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		border: none;
		border-radius: 999px;
		background: transparent;
		color: var(--gray-600, #4b5563);
		cursor: pointer;
		opacity: 0;
	}
	.swipe-delete-btn:hover {
		background: var(--gray-100, #f3f4f6);
		color: #dc2626;
	}
	/* Mouse: show on hover. Keyboard: show on focus. */
	@media (hover: hover) {
		.swipe-front:hover .swipe-delete-btn {
			opacity: 1;
		}
	}
	.swipe-front:focus-within .swipe-delete-btn,
	.swipe-delete-btn:focus-visible {
		opacity: 1;
	}
	@media (prefers-reduced-motion: reduce) {
		.swipe-row,
		.swipe-front,
		.swipe-front.swipe-leaving {
			transition: none;
		}
	}
</style>
