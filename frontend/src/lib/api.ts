/**
 * API facade — the stable `api.<resource>.<verb>()` surface the app imports.
 *
 * Each method delegates to an orval-generated function in
 * `./api/generated/go-stop-api.ts`. The generated functions go through the
 * `customFetch` mutator (`./api/fetchMutator.ts`), which prepends `/api`,
 * unwraps the JSON body, maps `{error}` -> ApiError, and turns 204 into null.
 *
 * Note on types: orval's `fetch` client types each generated function as
 * returning a `{ data, status, headers }` wrapper, but our mutator returns the
 * bare parsed body at runtime. So every call is cast (via `unwrap`) to the
 * friendly Appendix A body type. This keeps the downstream `api.*` signatures
 * exactly as the contract specifies.
 */
import { ApiError } from './api/fetchMutator';
import * as gen from './api/generated/go-stop-api';
import type {
	Ride,
	PublicRide,
	PublicRequest,
	Request,
	InterestListItem,
	MyInterest,
	ContactInfo,
	ExpressInterestResponse,
	AcceptInterestResponse,
	NotificationItem,
	Stats,
	Config,
	VapidKey,
	PostRideBody,
	PostRequestBody,
	SubscriptionBody,
	RideSearchParams
} from './types';

export { ApiError };

/**
 * The mutator returns the bare response body, but the generated functions are
 * typed as returning orval's `{ data, status, headers }` wrapper. Reconcile by
 * casting the awaited result to the friendly body type.
 */
function unwrap<T>(p: Promise<unknown>): Promise<T> {
	return p as Promise<T>;
}

/** X-Phone header for owner-scoped reads (passed via the generated `options` arg). */
function phoneHeader(phone: string): RequestInit {
	return { headers: { 'X-Phone': phone } };
}

export const api = {
	config: {
		get: () => unwrap<Config>(gen.getConfig())
	},
	vapid: {
		getPublicKey: () => unwrap<VapidKey>(gen.getVapidPublicKey())
	},
	stats: {
		get: () => unwrap<Stats>(gen.getStats())
	},
	destinations: {
		list: () => unwrap<string[]>(gen.listDestinations())
	},
	rides: {
		// feed (no args) | search (origin+destination [+date/time]) | my-rides (phone)
		list: (params: RideSearchParams = {}, phone?: string) =>
			unwrap<PublicRide[] | Ride[]>(gen.listRides(params, phone ? phoneHeader(phone) : undefined)),
		get: (id: string) => unwrap<Ride>(gen.getRide(id)),
		post: (body: PostRideBody) => unwrap<Ride>(gen.createRide(body)),
		del: (id: string, phone: string) => unwrap<null>(gen.deleteRide(id, { phone })),
		listInterests: (id: string, phone: string) =>
			unwrap<InterestListItem[]>(gen.listRideInterests(id, phoneHeader(phone))),
		listMatchingRequests: (id: string, phone: string) =>
			unwrap<PublicRequest[]>(gen.listRideRequests(id, phoneHeader(phone))),
		feedback: (id: string, phone: string, taken: boolean) =>
			unwrap<null>(gen.submitRideFeedback(id, { phone, taken }))
	},
	requests: {
		list: (phone: string) => unwrap<Request[]>(gen.listRequests(phoneHeader(phone))),
		get: (id: string, phone: string) => unwrap<Request>(gen.getRequest(id, phoneHeader(phone))),
		post: (body: PostRequestBody) => unwrap<Request>(gen.createRequest(body)),
		del: (id: string, phone: string) => unwrap<null>(gen.deleteRequest(id, { phone })),
		ping: (id: string, rideId: string, phone: string) =>
			unwrap<null>(gen.pingRequest(id, { ride_id: rideId }, phoneHeader(phone)))
	},
	interests: {
		express: (rideId: string, phone: string, name?: string) =>
			unwrap<ExpressInterestResponse>(gen.expressInterest(rideId, { phone, name })),
		accept: (interestId: string, phone: string) =>
			unwrap<AcceptInterestResponse>(gen.acceptInterest(interestId, { phone })),
		getContact: (interestId: string, phone: string) =>
			unwrap<ContactInfo>(gen.getInterestContact(interestId, phoneHeader(phone))),
		listMine: (phone: string) => unwrap<MyInterest[]>(gen.listMyInterests(phoneHeader(phone)))
	},
	subscriptions: {
		upsert: (body: SubscriptionBody) => unwrap<null>(gen.upsertSubscription(body)),
		remove: (phone: string) => unwrap<null>(gen.removeSubscription(encodeURIComponent(phone)))
	},
	notifications: {
		list: (phone: string) => unwrap<NotificationItem[]>(gen.listNotifications(phoneHeader(phone)))
	}
};
