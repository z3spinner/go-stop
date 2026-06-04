# AGENTS.md — `internal/boundaries/` (ports + HTTP adapters)

Two responsibilities live here:
1. `repository/` and `notification/` — the **interfaces** (ports) that `usecase`
   depends on and `infrastructure` implements. Pure interface declarations over
   `domain` types; no logic.
2. `handler/` — the **Gin HTTP adapters** that translate requests into use-case
   calls and results into JSON.

This package may import `domain`, `usecase`, and `gin` (handlers only). It must
**not** import `infrastructure`.

## Ports (`repository/`, `notification/`)

Each file declares one interface over domain types, e.g.:

```go
// notification/notifier.go
type Notifier interface {
    Send(subscription domain.Subscription, message domain.Message) error
}
```

Add a method here first when a use case needs new persistence/notification
behaviour, then implement it in `infrastructure` and run `make sqlc` if it's
backed by a new query.

## Handlers (`handler/`)

One handler struct per resource, constructed with the use cases and read-only
repos it needs; one method per endpoint with a `*gin.Context` receiver:

```go
func (h *RideHandler) Post(c *gin.Context) { /* bind → call use case → c.JSON */ }
```

Wiring: construct the handler in `main.go` and register its methods in the
`/api` route group there.

### OpenAPI (swaggo) annotations — required on every endpoint

Each handler method carries a swaggo doc block. This is the **source of the
OpenAPI spec and the generated frontend client** — keep it accurate.

```go
// Post creates a new ride offer.
// @ID       createRide
// @Tags     rides
// @Accept   json
// @Produce  json
// @Param    body  body  handler.PostRideBody  true  "Ride to create"
// @Success  201  {object}  domain.Ride
// @Failure  400  {object}  handler.ErrorResponse
// @Router   /rides [post]
func (h *RideHandler) Post(c *gin.Context) {
```

After adding or changing an endpoint or its DTOs, run **`make api-generate`**
(regenerates `docs/swagger.*` then the frontend `go-stop-api.ts`) and commit the
generated files. `make swagger` alone updates only the Go-side spec.

### DTOs (`dto.go`) and the PascalCase wire format

- **Request/response DTOs declared in `dto.go` carry lowercase `json` tags**
  (`json:"driver_name"`) — they define the exact wire format and exist mainly so
  swaggo has named schemas to reference.
- **Domain entities returned directly (`domain.Ride`, `domain.Request`) are
  tag-less and serialize as PascalCase** (`{"ID":..., "DriverName":...}`), matched
  by swaggo's `--propertyStrategy pascalcase`. Do not add `json` tags to domain
  types. When in doubt, check `docs/swagger.json` for the field casing a response
  actually uses.
- Map domain → DTO in the handler (helpers like `toPublicRides`).

### Input normalization & error mapping

- Normalize all phone and location inputs with `normalizePhone` /
  `normalizeLocation` (`phone.go`) before passing them on — this guarantees
  canonical storage and matching. (Route/location matching also folds accents,
  case, and whitespace by design but deliberately does not token- or fuzzy-match
  extra location words — see the project memory note before "improving" it.)
- Map errors to status codes by `errors.Is` on the use-case sentinels:
  `ErrUnauthorized` → 403, missing `X-Phone` header → 401, repo not-found → 404,
  bind failure → 400, `ErrNotPending` → 409, otherwise → 500. Use the
  `ErrorResponse` shape (`{"error": "..."}`).
