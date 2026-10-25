<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { formatTime, formatDate, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Request, ContactOffer } from '$lib/types';

	let { request, onseematches }: { request: Request; onseematches?: (o: string, d: string, dep: string) => void } = $props();
	let msg = $state('');
	let deleted = $state(false);
	let contactOffers = $state<ContactOffer[]>([]);

	const ZERO = '0001-01-01T00:00:00Z';
	const RENEW_WINDOW_MS = 7 * 24 * 60 * 60 * 1000;
	const hasDate = $derived(request.Date !== ZERO && request.Date?.slice(0, 4) !== '0001');
	const hasTime = $derived(request.DepartureAt !== ZERO && request.DepartureAt?.slice(0, 4) !== '0001');
	const isDaily = $derived(hasTime && request.DepartureAt.slice(0, 10) === '1970-01-01');
	// Open-ended alerts (anytime or daily) last 30 days unless renewed; the
	// owner is notified 3 days before. Renewing is offered in the last week.
	const isOpenEnded = $derived(!hasDate && (!hasTime || isDaily));
	let expiresAt = $state(untrack(() => request.ExpiresAt ?? ''));
	const expiringSoon = $derived.by(() => {
		const t = Date.parse(expiresAt);
		return !Number.isNaN(t) && t - Date.now() <= RENEW_WINDOW_MS;
	});
	let renewing = $state(false);

	async function renew() {
		if (renewing) return;
		renewing = true;
		try {
			const r = await api.requests.renew(request.ID, get(userPhone));
			expiresAt = r.ExpiresAt;
			msg = m.alertRenewed({ date: formatDate(r.ExpiresAt) });
		} catch {
			msg = m.alertRenewErr();
		} finally {
			renewing = false;
		}
	}

	async function del() {
		try {
			await api.requests.del(request.ID, get(userPhone));
			msg = m.deleteOk();
			deleted = true;
		} catch {
			msg = m.deleteErr();
		}
	}

	onMount(async () => {
		const phone = get(userPhone);
		if (phone) {
			contactOffers = await api.requests.listOffers(request.ID, phone).catch(() => []);
		}
	});
</script>

<div class="card rounded border p-3" id="card-{request.ID}" style:opacity={deleted ? 0.4 : 1}>
	<div class="card-route font-medium" translate="no">{request.Origin} <span class="route-arrow">→</span> {request.Destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		{#if !hasDate && !hasTime}
			<span class="tag tag-anytime">{m.alertAnytimeLabel()}</span>
		{:else if isDaily}
			<span class="tag tag-daily">{new Date(request.DepartureAt).toISOString().slice(11, 16)}</span>
			<span class="tag">{flexLabel(request.Flexibility)}</span>
		{:else if hasDate && !hasTime}
			<span class="tag">{formatDate(request.Date)}</span>
		{:else}
			<span>{formatTime(request.DepartureAt)}</span>
			<span class="tag">{flexLabel(request.Flexibility)}</span>
		{/if}
		{#if isOpenEnded && expiresAt && !deleted}
			<span class="tag tag-expires" class:tag-expiring={expiringSoon}>{m.alertExpiresOn({ date: formatDate(expiresAt) })}</span>
		{/if}
	</div>
	<div class="alert-actions flex gap-2">
		<button type="button" class="btn-see-matches" data-origin={request.Origin} data-dest={request.Destination}
			data-dept={hasTime ? request.DepartureAt : ''}
			onclick={() => onseematches?.(request.Origin, request.Destination, hasTime ? request.DepartureAt : '')}>{m.btnSeeMatches()}</button>
		{#if isOpenEnded && expiringSoon && !deleted}
			<button type="button" class="btn-renew" data-id={request.ID} disabled={renewing} onclick={renew}>{m.btnRenewAlert()}</button>
		{/if}
		<button type="button" class="btn btn-danger btn-delete" data-id={request.ID} data-phone={get(userPhone)} disabled={deleted} onclick={del}>{m.btnDelete()}</button>
	</div>
	<div class="delete-msg" id="msg-{request.ID}">{msg}</div>
	{#if contactOffers.length > 0}
		<div class="contact-offers mt-2 border-t pt-2">
			<div class="contact-offers-title mb-1 text-xs font-semibold text-gray-500">{m.contactOffersTitle()} ({contactOffers.length})</div>
			<ul class="contact-offers-list space-y-1">
				{#each contactOffers as offer (offer.id)}
					<li class="contact-offer-item text-sm">
						<span class="font-medium">{offer.offerer_name}</span>
						— <a class="underline underline-offset-2" href="tel:{offer.offerer_phone}">{offer.offerer_phone}</a>
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</div>
