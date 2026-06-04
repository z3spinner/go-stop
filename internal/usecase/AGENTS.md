# AGENTS.md — `internal/usecase/` (business logic)

One use case per file, each a struct of injected interfaces plus a single
`Execute` method. This is the only layer that orchestrates domain logic.

## The pattern

```go
type ExpressInterest struct {
    rides     repository.RideRepository      // interfaces from internal/boundaries
    interests repository.InterestRepository
    subs      repository.SubscriptionRepository
    notifier  notification.Notifier
}

func NewExpressInterest(rides repository.RideRepository, /* ... */) *ExpressInterest {
    return &ExpressInterest{rides: rides, /* ... */}
}

func (uc *ExpressInterest) Execute(rideID, searcherPhone, searcherName string) (domain.Interest, error) {
    // validate → call repositories → mutate domain → notify → return
}
```

Rules:
- **Depend only on `domain` and the `boundaries` interfaces** (plus stdlib).
  Importing `infrastructure`, `pgx`, or `gin` here breaks the architecture.
- Constructor `NewXxx` takes every dependency as a parameter — no globals, no
  singletons. `main.go` wires the concretes in.
- Keep one public entry point (`Execute`); even trivial passthrough use cases
  (`GetRides`, `Subscribe`) follow the full struct+constructor shape for
  consistency and testability.
- Return `domain` types and `error`. Use the sentinel errors in `errors.go`
  (`ErrUnauthorized`, `ErrNotPending`) for conditions handlers must distinguish.

## Shared helpers

- `match.go` — `WindowsOverlap(ride, req)`: symmetric time-window overlap; each
  side widens its `DepartureAt` by its own `Flexibility` minutes. Pure function,
  the heart of matching.
- `notify.go` — `sendToAll`, `NotifySearcher`, `NotifyDriver`: best-effort push
  fan-out to a phone's devices; prunes subscriptions that return `410 Gone`.
  Reuse these instead of calling `notifier.Send` directly from a new use case.

Authorization is done **inside** the use case (e.g. `DeleteRide`/`AcceptInterest`
compare the caller's phone to the ride's `Phone` and return `ErrUnauthorized`),
not in the handler.

## Testing — unit tests, no DB, no build tag

`*_test.go` files here are plain `go test ./internal/usecase/...` (`make test-unit`),
**no `integration` tag**. Conventions:

- Package `usecase_test` (black-box).
- **Hand-written mocks, no mocking library.** Mocks implement the `boundaries`
  interfaces and are defined in the test files; shared ones (`mockRideRepo`,
  `mockRequestRepo`, `mockSubRepo`, `mockNotifier`) live alongside the first test
  that needed them and are reused — search before adding a duplicate.
- One test function per scenario, named `Test<UseCase>_<Scenario>`; assert on
  returned values and on recorded mock state (e.g. `notifier.sent`, `repo.saved`).
  Check sentinel errors with `errors.Is`.

When adding a use case: write the struct + `NewXxx` + `Execute`, add a `*_test.go`
with mocks, then wire it into `main.go` and (usually) a handler.
