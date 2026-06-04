# True One-Click Scalingo Deploy

**Date:** 2026-06-04
**Status:** Approved design, pending implementation plan

## Goal

Make the Scalingo "Deploy" button a genuine one-click experience: a new community
can spin up its own Go-Stop instance without generating or pasting any VAPID keys.
A "Launch your own community" entry point on the running site links to that deploy
flow.

## Background / constraint

Today `scalingo.json` marks `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, and
`VAPID_EMAIL` as `required: true`. Scalingo renders an empty, mandatory form field
for each, so the deployer must first generate an EC P-256 keypair by hand and paste
it in. That is the only thing breaking one-click.

Confirmed from Scalingo's docs:

- The deploy URL accepts only `source=<git-repo>#<branch>`. There is **no** way to
  inject env-var *values* through the link.
- All env defaults come from the `scalingo.json` committed in the source repo.
- The deploy flow always shows a pre-filled confirmation form; "one-click" means
  every field is already valid so the deployer just clicks Deploy.
- `scalingo.json` generators (`secret`, `template`, `url`) produce only random
  tokens / string templates — none can produce a corresponding EC keypair.

Therefore the new instance must provision its own VAPID keypair. The DB is the
single runtime source of truth; env vars are a one-time bootstrap seed for
continuity with existing deployments.

## Approach

On first boot, if no keypair exists in the database, the app generates one (or
adopts keys already supplied via env), persists it, and uses it thereafter. The
VAPID env vars are removed from `scalingo.json` entirely, so the deploy form has
zero required fields.

### Resolution order (runs once per boot, in `main.go`)

1. Read `vapid_public` / `vapid_private` / `vapid_email` from the `app_settings`
   table. If present → use them. **(Steady state — DB is the only source.)**
2. If absent **and** `VAPID_PUBLIC_KEY` + `VAPID_PRIVATE_KEY` env vars are set →
   **adopt** them into `app_settings` (preserves existing push subscribers, whose
   subscriptions are bound to the public key), then use them.
3. If absent and no env → generate a fresh keypair via
   `webpush.GenerateVAPIDKeys()`, persist, use.

After step 2 or 3 runs once, env vars are never read again. `VAPID_EMAIL` falls
back to a sane default (`mailto:admin@example.com`) when neither DB nor env
supplies one; push services accept it and the value can be changed later by
writing `app_settings`.

The path taken is logged at boot (`vapid: loaded from db` / `adopted from env` /
`generated fresh`).

## Components

### 1. Migration `011_app_settings`

Generic key/value table (reusable for future auto-config, not VAPID-specific):

```sql
CREATE TABLE app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`.up.sql` and `.down.sql` (drop table) following the existing
`internal/infrastructure/postgres/sqlc/migrations/NNN_name.{up,down}.sql`
convention. Applied by `migratedb up` in the `web` command (`web: migratedb up
&& go-stop`) so the table exists before the app reads it at boot. NOTE: on
Scalingo the `postdeploy` hook runs *after* the web container boots, so a
migration placed only in `postdeploy` would not exist yet when `vapid.Resolve`
queries it — the migration must run as part of the web command.

### 2. Settings repo (`internal/infrastructure/postgres`)

- `Get(ctx, key) (value string, found bool, err error)`
- `InsertIfAbsent(ctx, key, value) error` — `INSERT … ON CONFLICT (key) DO NOTHING`.

These two primitives make provisioning race-safe across multiple web containers:
each writer does `InsertIfAbsent` for all three keys, then everyone re-reads; the
loser of the race reads the winner's keys.

### 3. VAPID resolution helper

`resolveVAPID(ctx, settingsRepo) (public, private, email string, err error)`,
implementing the resolution order above. Keypair generation via the existing
`SherClockHolmes/webpush-go` dependency (`GenerateVAPIDKeys()`).

### 4. `main.go` wiring

Replace the three direct env reads:

- `webpush.New(os.Getenv("VAPID_PUBLIC_KEY"), …)` → `webpush.New(pub, priv, email)`
- `handler.NewVapidHandler(os.Getenv("VAPID_PUBLIC_KEY"))` → `handler.NewVapidHandler(pub)`

with the resolved values. Generation or persistence failure is fatal with a clear
message (the app cannot send push without a key; this should effectively never
happen given crypto-rand).

### 5. `scalingo.json`

Remove `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, and `VAPID_EMAIL` from the `env`
block. Remaining vars (`SITE_NAME`, `SERVICE_TZ`, `RIDE_GRACE_MINUTES`,
`RETURN_DELAY_HOURS`, `GIN_MODE`) all have defaults, so no field is required → the
deploy form is click-through.

### 6. In-site "Launch your own community" button

Add to `frontend/src/lib/components/layout/AboutModal.svelte` (where GitHub/about
content already lives) a link/button to:

```
https://my.scalingo.com/deploy?source=https://github.com/z3spinner/go-stop
```

i18n strings added across all existing locales: `en, es, de, nl, fr, it`.

### 7. Doc cleanup

- README "Local setup (manual)" and "Docker" sections: note that VAPID keys
  auto-generate on first boot if unset; env vars remain supported for explicit
  local control.
- `.env.example`: keep VAPID entries but mark optional with a note that they
  auto-generate when blank.
- Remove the dangling `go run ./cmd/genvapid` reference (README); the matching
  `scalingo.json` reference disappears when the VAPID env block is removed.
  (Adding a real `cmd/genvapid` convenience command is optional, not required.)

## Data flow

```
boot
 └─ postgres.NewPool()
     └─ settingsRepo
         └─ resolveVAPID()  →  (public, private, email)
             ├─ webpush.New(public, private, email)   // notifier
             └─ handler.NewVapidHandler(public)        // GET public key for browsers
```

The browser-facing VAPID public-key endpoint is unchanged; it now serves the
resolved key rather than the raw env var.

## Error handling

- `GenerateVAPIDKeys()` failure → fatal, clear log (essentially impossible).
- `app_settings` write failure during provisioning → fatal, clear log (persistence
  is required for stable keys across restarts).
- Multi-container first-boot race → handled by `InsertIfAbsent` + re-read.

## Testing

Integration tests (tagged, against the test DB) for `resolveVAPID` + settings repo:

1. Empty DB, no env → generates and persists; a second call returns the **same**
   keys (stability).
2. Empty DB, env set → adopts env values into the DB; with env later cleared, the
   same keys still return from the DB.
3. DB already populated → returns DB keys and ignores differing env values.

Frontend: assert the AboutModal renders the launch button with the correct
`https://my.scalingo.com/deploy?source=…` href.

## Out of scope

- `PHONE_ENCRYPTION_KEY` (not implemented in the codebase; explicitly ignored).
- Programmatic Scalingo-API provisioning / white-label portal.
- Baked per-deploy source branches.
- A guided post-deploy setup/onboarding page.
