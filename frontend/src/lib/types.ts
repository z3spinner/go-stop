// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Friendly domain types for the app.
 *
 * The shapes themselves are generated from the Go OpenAPI spec by orval
 * (`./api/generated/go-stop-api.ts`). This module re-exports those generated
 * models under the friendly Appendix A names (`Ride`, `PublicRide`, …) so
 * downstream components import stable names while the underlying shapes stay
 * correct-by-construction with the backend.
 *
 * Response entity types are wrapped in `DeepRequired` so that component code
 * can rely on all server-populated fields being present (the OpenAPI spec
 * marks nothing `required`, but the server always populates these fields).
 *
 * `Flexibility` is hand-defined (a 0 | 30 | 60 union) independent of the
 * generated `DomainFlexibility` so call sites get the precise literal union.
 */

// Flexibility is a numeric enum: 0 = exact, 30 = ±30 min, 60 = ±60 min.
export type Flexibility = 0 | 30 | 60;

/**
 * Recursively strips `?` (optional / undefined) from every property of T,
 * including nested objects and array element types.
 */
type DeepRequired<T> = T extends (infer U)[]
	? DeepRequired<U>[]
	: T extends object
		? { [K in keyof T]-?: DeepRequired<T[K]> }
		: T;

// ── Response entity imports ──────────────────────────────────────────────────
// Each of these is fully populated by the server on every response, so we wrap
// them in DeepRequired to eliminate spurious `| undefined` at call sites.

import type {
	DomainRide,
	HandlerPublicRide,
	HandlerPublicRequest,
	DomainRequest,
	HandlerInterestListItem,
	HandlerMyInterest,
	HandlerContactInfo,
	HandlerExpressInterestResponse,
	HandlerAcceptInterestResponse,
	HandlerNotificationItem,
	DomainStats,
	DomainRouteStat,
	DomainActivityCounts,
	HandlerConfigResponse,
	HandlerVapidKeyResponse,
	// Request bodies / params — keep as-is (callers build these, optionality is fine)
	HandlerPostRideBody as PostRideBody,
	HandlerUpdateRideBody as UpdateRideBody,
	HandlerPostRequestBody as PostRequestBody,
	HandlerSubscriptionBody as SubscriptionBody,
	ListRidesParams as RideSearchParams
} from './api/generated/go-stop-api';

// Re-export request body / param aliases unchanged
export type { PostRideBody, UpdateRideBody, PostRequestBody, SubscriptionBody, RideSearchParams };

// ── Response entity aliases (all fields required) ────────────────────────────
export type Ride = DeepRequired<DomainRide>;
export type PublicRide = DeepRequired<HandlerPublicRide>;
export type PublicRequest = DeepRequired<HandlerPublicRequest>;
export type Request = DeepRequired<DomainRequest>;
export type MyInterest = DeepRequired<HandlerMyInterest>;
export type ContactInfo = DeepRequired<HandlerContactInfo>;
export type ExpressInterestResponse = DeepRequired<HandlerExpressInterestResponse>;
export type AcceptInterestResponse = DeepRequired<HandlerAcceptInterestResponse>;
export type NotificationItem = DeepRequired<HandlerNotificationItem>;
export type Stats = DeepRequired<DomainStats>;
export type RouteStat = DeepRequired<DomainRouteStat>;
export type ActivityCounts = DeepRequired<DomainActivityCounts>;
export type Config = DeepRequired<HandlerConfigResponse>;
export type VapidKey = DeepRequired<HandlerVapidKeyResponse>;

// ── InterestListItem — partial exception ─────────────────────────────────────
// `id` and `status` are always populated; `searcher_name` / `searcher_phone`
// are `omitempty` on the server and may genuinely be absent.
export type InterestListItem = DeepRequired<Pick<HandlerInterestListItem, 'id' | 'status'>> &
	Pick<HandlerInterestListItem, 'searcher_name' | 'searcher_phone'>;

// ContactOffer is one entry of GET /requests/{id}/offers.
export type ContactOffer = {
	id: string;
	offerer_name: string;
	offerer_phone: string;
};
