/**
 * Friendly domain types for the app.
 *
 * The shapes themselves are generated from the Go OpenAPI spec by orval
 * (`./api/generated/go-stop-api.ts`). This module re-exports those generated
 * models under the friendly Appendix A names (`Ride`, `PublicRide`, …) so
 * downstream components import stable names while the underlying shapes stay
 * correct-by-construction with the backend.
 *
 * `Flexibility` is hand-defined (a 0 | 30 | 60 union) independent of the
 * generated `DomainFlexibility` so call sites get the precise literal union.
 */

// Flexibility is a numeric enum: 0 = exact, 30 = ±30 min, 60 = ±60 min.
export type Flexibility = 0 | 30 | 60;

export type {
	// Entities
	DomainRide as Ride,
	HandlerPublicRide as PublicRide,
	HandlerPublicRequest as PublicRequest,
	DomainRequest as Request,
	HandlerInterestListItem as InterestListItem,
	HandlerMyInterest as MyInterest,
	HandlerContactInfo as ContactInfo,
	HandlerExpressInterestResponse as ExpressInterestResponse,
	HandlerAcceptInterestResponse as AcceptInterestResponse,
	HandlerNotificationItem as NotificationItem,
	DomainStats as Stats,
	DomainRouteStat as RouteStat,
	DomainActivityCounts as ActivityCounts,
	HandlerConfigResponse as Config,
	HandlerVapidKeyResponse as VapidKey,
	// Request bodies / params
	HandlerPostRideBody as PostRideBody,
	HandlerPostRequestBody as PostRequestBody,
	HandlerSubscriptionBody as SubscriptionBody,
	ListRidesParams as RideSearchParams
} from './api/generated/go-stop-api';
