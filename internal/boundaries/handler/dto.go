package handler

import "time"

// This file defines the named request/response DTO structs referenced by the
// swag (OpenAPI) annotations on the handlers. They mirror EXACTLY the JSON shape
// each handler already emits/binds — they exist so swag can name and shape the
// generated schema. They are not used by the runtime handlers (which still bind
// their own private request structs and emit gin.H / inline objects), except
// where a handler explicitly serialises one of these types.

// ErrorResponse is the standard JSON error envelope ({"error": "..."}).
type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Request bodies ---

// PostRideBody is the body for POST /rides.
type PostRideBody struct {
	DriverName  string `json:"driver_name"`
	Phone       string `json:"phone"`
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	DepartureAt string `json:"departure_at"`
	Flexibility int    `json:"flexibility"`
}

// DeleteRideBody is the body for DELETE /rides/{id} (owner phone).
type DeleteRideBody struct {
	Phone string `json:"phone"`
}

// ExpressInterestBody is the body for POST /rides/{id}/interest.
type ExpressInterestBody struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

// FeedbackBody is the body for POST /rides/{id}/feedback.
type FeedbackBody struct {
	Phone string `json:"phone"`
	Taken bool   `json:"taken"`
}

// AcceptInterestBody is the body for POST /interests/{id}/accept (driver phone).
type AcceptInterestBody struct {
	Phone string `json:"phone"`
}

// PostRequestBody is the body for POST /requests.
type PostRequestBody struct {
	SearcherName  string `json:"searcher_name"`
	Phone         string `json:"phone"`
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureAt   string `json:"departure_at"`   // RFC3339 — specific time mode
	DepartureDate string `json:"departure_date"` // YYYY-MM-DD — day mode
	DepartureTime string `json:"departure_time"` // HH:MM — daily mode
	Flexibility   int    `json:"flexibility"`
}

// DeleteRequestBody is the body for DELETE /requests/{id} (owner phone).
type DeleteRequestBody struct {
	Phone string `json:"phone"`
}

// PingRequestBody is the body for POST /requests/{id}/ping.
type PingRequestBody struct {
	RideID string `json:"ride_id"`
}

// SubscriptionBody is the body for POST /subscriptions.
type SubscriptionBody struct {
	Phone    string `json:"phone"`
	Endpoint string `json:"endpoint"`
	P256DH   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

// --- Response shapes ---

// PublicRide is returned for public search/feed requests. Phone is absent; DriverName is visible.
type PublicRide struct {
	ID            string    `json:"ID"`
	DriverName    string    `json:"DriverName"`
	Origin        string    `json:"Origin"`
	Destination   string    `json:"Destination"`
	Date          time.Time `json:"Date"`
	DepartureAt   time.Time `json:"DepartureAt"`
	Flexibility   int       `json:"Flexibility"`
	PostedAt      time.Time `json:"PostedAt"`
	ExpiresAt     time.Time `json:"ExpiresAt"`
	FeedbackGiven bool      `json:"FeedbackGiven"`
	InterestCount int       `json:"InterestCount"`
}

// PublicRequest is a phone-stripped request matched to a driver's ride
// (GET /rides/{id}/requests). PascalCase to match the wire format.
type PublicRequest struct {
	ID           string    `json:"ID"`
	SearcherName string    `json:"SearcherName"`
	Origin       string    `json:"Origin"`
	Destination  string    `json:"Destination"`
	DepartureAt  time.Time `json:"DepartureAt"`
	Flexibility  int       `json:"Flexibility"`
}

// InterestListItem is one entry of GET /rides/{id}/interests (driver's view).
type InterestListItem struct {
	SearcherName  string `json:"searcher_name,omitempty"`
	ID            string `json:"id"`
	Status        string `json:"status"`
	SearcherPhone string `json:"searcher_phone,omitempty"`
}

// MyInterest is one entry of GET /interests (searcher's own contact requests).
type MyInterest struct {
	ID          string `json:"id"`
	RideID      string `json:"ride_id"`
	Status      string `json:"status"`
	DriverName  string `json:"driver_name"`
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	DepartureAt string `json:"departure_at"`
}

// ExpressInterestResponse is returned by POST /rides/{id}/interest.
type ExpressInterestResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// AcceptInterestResponse is returned by POST /interests/{id}/accept.
type AcceptInterestResponse struct {
	SearcherPhone string `json:"searcher_phone"`
}

// ContactInfo is returned by GET /interests/{id}/contact (accepted mutual contact).
type ContactInfo struct {
	Phone       string    `json:"phone"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	DepartureAt time.Time `json:"departure_at"`
}

// NotificationItem is one entry of GET /notifications.
type NotificationItem struct {
	RideID      string `json:"ride_id"`
	RequestID   string `json:"request_id"`
	DriverName  string `json:"driver_name"`
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	DepartureAt string `json:"departure_at"`
	SentCount   int    `json:"sent_count"`
}

// ConfigResponse is returned by GET /config.
type ConfigResponse struct {
	SiteName         string `json:"siteName"`
	ReturnDelayHours int    `json:"returnDelayHours"`
}

// VapidKeyResponse is returned by GET /vapid-public-key.
type VapidKeyResponse struct {
	PublicKey string `json:"publicKey"`
}
