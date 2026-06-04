# AGENTS.md — `internal/` (Clean Architecture core)

Four layers, dependencies pointing **inward only**. Never import outward.

```
domain  ←  usecase  ←  boundaries  ←  infrastructure
(types)    (logic)     (ports)         (adapters)
```

| Layer | Imports allowed | Must NOT import |
|---|---|---|
| `domain` | stdlib only (`time`, etc.) | anything in this repo |
| `usecase` | `domain`, `boundaries/*` interfaces, stdlib | `infrastructure`, `gin`, `pgx` |
| `boundaries` | `domain`, `usecase`, stdlib, `gin` (handlers only) | `infrastructure` |
| `infrastructure` | everything (it's the outer ring) | — |

The dependency rule is the one invariant to protect. If you find yourself
importing `infrastructure` or a driver (`pgx`, `gin`, `webpush`) from `usecase`,
stop — you need an interface in `boundaries` instead.

## Where interfaces live (this is the key idea)

Ports are **defined in `boundaries`**, not where they're used or implemented:

- `boundaries/repository/*.go` — repository interfaces (`RideRepository`,
  `InterestRepository`, `SubscriptionRepository`, `NotificationQueueRepository`,
  `StatRepository`, `DestinationRepository`, `SettingsRepository`).
- `boundaries/notification/notifier.go` — `Notifier` (single `Send` method).

They are **consumed by `usecase`** (constructor parameters) and **implemented by
`infrastructure`** (`postgres`, `webpush`). `main.go` is the only place the
concrete types meet the interfaces.

## domain — pure types

Anemic, exported structs and value objects: `Ride`, `Request`, `Interest`,
`Subscription`, `Message`, `Stats`, `NotificationQueueEntry`, plus the
`Flexibility` value object (`int` minutes: `0` Exact / `30` Approximate / `60`
Flexible). Conventions:

- **No external dependencies. No business behaviour** — entities are data;
  logic lives in `usecase`. Matching (`usecase/match.go`'s `WindowsOverlap`)
  reads domain fields but is not a method on them.
- **No struct tags on entities** — `domain.Ride`/`domain.Request` are tag-less.
  They serialize to JSON as **PascalCase** field names (`{"ID":..., "Origin":...}`)
  because swaggo's `--propertyStrategy pascalcase` documents them that way and Gin
  emits exported field names verbatim. Do not add `json` tags to these — it would
  change the wire format and desync the generated frontend client.
  (Boundary DTOs in `boundaries/handler/dto.go` are the exception: they *do*
  carry lowercase `json` tags. See `internal/boundaries/AGENTS.md`.)
- Zero values are meaningful: a zero `Date`/`DepartureAt` means "no constraint";
  `DepartureAt` set to `1970-01-01` is the sentinel for daily/recurring alerts.

## Cross-cutting conventions

- One concept per file, named after it (`express_interest.go`, `ride_repo.go`).
- Errors that callers branch on are **sentinel errors** (`usecase/errors.go`:
  `ErrUnauthorized`, `ErrNotPending`), checked with `errors.Is`. Handlers map
  them to HTTP status codes; don't string-match error text.
- `context.Context` is created inside repository implementations
  (`context.Background()`), not threaded through use case signatures.
- Code must pass `make lint` (golangci-lint; config in `/.golangci.yml`) and
  `make fmt` (gofmt + goimports). The lint set is curated and currently clean —
  keep it that way. `misspell` is deliberately off because of the intentional
  non-English string literals (`usecase/qotd.go`, French push copy).
