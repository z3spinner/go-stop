# Go-Stop — Architecture & Design

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go |
| HTTP Framework | Gin (gin-gonic) |
| Database | PostgreSQL (Scalingo managed add-on) |
| Frontend | HTML / CSS / JS (served by Go) |
| Push Notifications | Web Push API (VAPID) |
| Hosting | Scalingo |

All layers are served from a single Go application — no separate frontend deployment.

---

## Architecture

The project follows **Uncle Bob's Clean Architecture**. Dependencies point inward only — outer layers know about inner layers, never the reverse.

```
┌─────────────────────────────────────┐
│         infrastructure              │  PostgreSQL, Web Push
│  ┌───────────────────────────────┐  │
│  │         boundaries            │  │  Gin handlers, repository interfaces
│  │  ┌─────────────────────────┐  │  │
│  │  │        usecase          │  │  │  Business logic
│  │  │  ┌───────────────────┐  │  │  │
│  │  │  │      domain       │  │  │  │  Entities, types
│  │  │  └───────────────────┘  │  │  │
│  │  └─────────────────────────┘  │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

---

## Project Structure

```
/
├── main.go
├── scalingo.json
├── Procfile
├── go.mod
├── go.sum
├── web/                    # Static frontend files
│   ├── index.html
│   ├── css/
│   └── js/
├── internal/
│   ├── domain/             # Core business entities
│   │   ├── ride.go
│   │   ├── request.go
│   │   ├── subscription.go
│   │   ├── message.go
│   │   └── flexibility.go
│   ├── usecase/            # Business logic
│   │   ├── post_ride.go
│   │   ├── post_request.go
│   │   ├── search_rides.go
│   │   ├── get_rides.go
│   │   ├── get_destinations.go
│   │   ├── match.go
│   │   ├── notify.go
│   │   ├── subscribe.go
│   │   ├── delete_ride.go
│   │   ├── delete_request.go
│   │   └── expire.go
│   ├── boundaries/
│   │   ├── handler/        # Gin HTTP handlers
│   │   │   ├── ride_handler.go
│   │   │   ├── request_handler.go
│   │   │   ├── search_handler.go
│   │   │   ├── destination_handler.go
│   │   │   └── subscription_handler.go
│   │   ├── repository/     # Repository interfaces
│   │   │   ├── ride_repository.go
│   │   │   ├── request_repository.go
│   │   │   ├── destination_repository.go
│   │   │   └── subscription_repository.go
│   │   └── notification/   # Notification interface
│   │       └── notifier.go
│   └── infrastructure/
│       ├── postgres/       # PostgreSQL implementations
│       │   ├── ride_repo.go
│       │   ├── request_repo.go
│       │   └── subscription_repo.go
│       └── webpush/        # Web Push implementation
│           └── webpush.go
└── docs/
    ├── requirements.md
    ├── design.md
    └── data-model.md
```

---

## Domain Layer

### Flexibility

```go
type Flexibility int

const (
    Exact       Flexibility = 0   // no flexibility
    Approximate Flexibility = 30  // ±30 minutes
    Flexible    Flexibility = 60  // ±60 minutes
)
```

Presets are presented as UI options; users can also enter a custom value in minutes.

### Ride

```go
type Ride struct {
    ID          string
    DriverName  string
    Phone       string
    Origin      string
    Destination string
    Date        time.Time
    DepartureAt time.Time
    Flexibility Flexibility
    PostedAt    time.Time
    ExpiresAt   time.Time
}
```

### Request

```go
type Request struct {
    ID           string
    SearcherName string
    Phone        string
    Origin       string
    Destination  string
    Date         time.Time
    DepartureAt  time.Time
    Flexibility  Flexibility
    PostedAt     time.Time
    ExpiresAt    time.Time
}
```

### Subscription

```go
type Subscription struct {
    ID       string
    Phone    string
    Endpoint string
    Keys     PushKeys
}

type PushKeys struct {
    P256DH string
    Auth   string
}
```

### Message

```go
type Message struct {
    Title       string
    Body        string
    URL         string
    ContactName string
    Phone       string
    Origin      string
    Destination string
    DepartureAt time.Time
}
```

---

## Use Cases

| Use Case | Description |
|---|---|
| `PostRide` | Save a ride, trigger matching, notify matching searchers |
| `PostRequest` | Save a request, trigger matching, notify matching drivers |
| `SearchRides` | Find rides by origin, destination, date, and flexibility window |
| `GetRides` | Browse all active rides (homepage feed) |
| `GetDestinations` | Return distinct origins and destinations for autocomplete |
| `MatchRequestsToRide` | Find requests that overlap with a given ride |
| `MatchRidesToRequest` | Find rides that overlap with a given request |
| `NotifyDriver` | Send push notification to a driver about a matching request |
| `NotifySearcher` | Send push notification to a searcher about a matching ride |
| `Subscribe` | Register a Web Push subscription linked to a phone number |
| `DeleteRide` | Delete a ride, authenticated by phone number |
| `DeleteRequest` | Delete a request, authenticated by phone number |
| `ExpireRides` | Delete all rides past their ExpiresAt (cron job) |
| `ExpireRequests` | Delete all requests past their ExpiresAt (cron job) |

---

## Boundaries Layer

### Repository Interfaces

```go
type RideRepository interface {
    Save(ride domain.Ride) error
    FindByID(id string) (domain.Ride, error)
    FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error)
    FindMatching(request domain.Request) ([]domain.Ride, error)
    Delete(id string) error
    DeleteExpired() error
}

type RequestRepository interface {
    Save(request domain.Request) error
    FindByID(id string) (domain.Request, error)
    FindMatching(ride domain.Ride) ([]domain.Request, error)
    Delete(id string) error
    DeleteExpired() error
}

type DestinationRepository interface {
    GetAll() ([]string, error)
}

type SubscriptionRepository interface {
    Save(subscription domain.Subscription) error
    FindByPhone(phone string) (domain.Subscription, error)
    Delete(phone string) error
}
```

### Notification Interface

```go
type Notifier interface {
    Send(subscription domain.Subscription, message domain.Message) error
}
```

---

## HTTP API

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/rides` | Post a new ride |
| `GET` | `/rides` | Browse rides (origin, destination, date as query params) |
| `GET` | `/rides/:id` | Get a specific ride (used by notification deep link) |
| `DELETE` | `/rides/:id` | Delete a ride (phone in body as auth) |
| `POST` | `/requests` | Post a new request |
| `GET` | `/requests/:id` | Get a specific request (used by notification deep link) |
| `DELETE` | `/requests/:id` | Delete a request (phone in body as auth) |
| `GET` | `/destinations` | Get all known origins and destinations |
| `POST` | `/subscriptions` | Register push notification subscription |
| `DELETE` | `/subscriptions/:phone` | Unsubscribe from push notifications |

Static files are served from the `/web` directory by the same Go application.

---

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string (set by Scalingo) | Yes |
| `VAPID_PUBLIC_KEY` | VAPID public key for Web Push | Yes |
| `VAPID_PRIVATE_KEY` | VAPID private key for Web Push | Yes |
| `VAPID_EMAIL` | Contact email for Web Push (e.g. mailto:you@example.com) | Yes |
| `PORT` | HTTP port (set by Scalingo) | Yes |
