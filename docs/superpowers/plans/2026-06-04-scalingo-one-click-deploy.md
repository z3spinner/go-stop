# True One-Click Scalingo Deploy — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A fresh Go-Stop instance provisions its own VAPID keypair on first boot, so the Scalingo deploy button needs zero manual input.

**Architecture:** A generic `app_settings` key/value table is the single runtime source of truth for VAPID keys. On boot, `vapid.Resolve` reads the keys from the DB; if absent, it adopts keys from env vars (one-time, for existing deployments) or generates a fresh keypair, persists them race-safely, and returns them. `main.go` wires the resolved keys into the push notifier and the public-key handler. The VAPID env vars are removed from `scalingo.json` so the deploy form has no required fields.

**Tech Stack:** Go, pgx/v5, sqlc, golang-migrate, `github.com/SherClockHolmes/webpush-go` (already a dependency, exposes `GenerateVAPIDKeys() (privateKey, publicKey string, err error)`).

**Already done (no task needed):** The in-site "Deploy on Scalingo" link already exists in the translated `aboutBody` message (`frontend/src/messages/*.json`) pointing at `https://my.scalingo.com/deploy?source=https://github.com/z3spinner/go-stop`. The spec's "in-site button" component is therefore already satisfied.

---

## File Structure

- **Create** `internal/infrastructure/postgres/sqlc/migrations/011_app_settings.up.sql` — table DDL.
- **Create** `internal/infrastructure/postgres/sqlc/migrations/011_app_settings.down.sql` — drop table.
- **Create** `internal/infrastructure/postgres/sqlc/queries/sql/settings.sql` — sqlc queries (Get / InsertIfAbsent).
- **Generated (via `make sqlc`)** `internal/infrastructure/postgres/sqlc/queries/settings.sql.go` + updates to `models.go`, `querier.go`.
- **Create** `internal/infrastructure/postgres/settings_repo.go` — `SettingsRepo` wrapping the generated queries.
- **Create** `internal/infrastructure/postgres/settings_repo_integration_test.go` — DB-backed test for the repo.
- **Create** `internal/infrastructure/vapid/resolve.go` — `Resolve` + `Store` interface + `Keys` struct. Pure logic, DB-agnostic.
- **Create** `internal/infrastructure/vapid/resolve_test.go` — unit tests with an in-memory fake store.
- **Modify** `main.go` — build `SettingsRepo`, call `vapid.Resolve`, wire keys into notifier + vapid handler.
- **Modify** `scalingo.json` — remove the three VAPID env entries.
- **Modify** `README.md` and `.env.example` — note auto-generation.

---

## Task 1: `app_settings` migration

**Files:**
- Create: `internal/infrastructure/postgres/sqlc/migrations/011_app_settings.up.sql`
- Create: `internal/infrastructure/postgres/sqlc/migrations/011_app_settings.down.sql`

- [ ] **Step 1: Write the up migration**

`internal/infrastructure/postgres/sqlc/migrations/011_app_settings.up.sql`:

```sql
-- Generic key/value store for runtime-provisioned configuration
-- (e.g. self-generated VAPID keys). Single source of truth at runtime.
CREATE TABLE app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- [ ] **Step 2: Write the down migration**

`internal/infrastructure/postgres/sqlc/migrations/011_app_settings.down.sql`:

```sql
DROP TABLE app_settings;
```

- [ ] **Step 3: Apply the migration to the local dev DB and verify the table exists**

Run (DATABASE_URL points at your local dev/test Postgres, e.g. the docker-compose one):

```bash
DATABASE_URL="postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" go run ./cmd/migratedb up
DATABASE_URL="postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" psql "$DATABASE_URL" -c '\d app_settings'
```

Expected: `Migration complete`, then a table description listing columns `key`, `value`, `created_at`.

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/postgres/sqlc/migrations/011_app_settings.up.sql internal/infrastructure/postgres/sqlc/migrations/011_app_settings.down.sql
git commit -m "feat(deploy): add app_settings table for runtime config"
```

---

## Task 2: Settings repo (sqlc queries + wrapper)

**Files:**
- Create: `internal/infrastructure/postgres/sqlc/queries/sql/settings.sql`
- Create: `internal/infrastructure/postgres/settings_repo.go`
- Test: `internal/infrastructure/postgres/settings_repo_integration_test.go`

- [ ] **Step 1: Write the failing integration test**

`internal/infrastructure/postgres/settings_repo_integration_test.go`:

```go
//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func clearSettings(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `DELETE FROM app_settings`); err != nil {
		t.Fatalf("clear app_settings: %v", err)
	}
}

func TestSettingsRepo_GetMissingReturnsNotFound(t *testing.T) {
	clearSettings(t)
	repo := postgres.NewSettingsRepo(testPool)

	_, found, err := repo.Get(context.Background(), "nope")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("expected found=false for missing key")
	}
}

func TestSettingsRepo_InsertIfAbsentThenGet(t *testing.T) {
	clearSettings(t)
	repo := postgres.NewSettingsRepo(testPool)
	ctx := context.Background()

	if err := repo.InsertIfAbsent(ctx, "k", "first"); err != nil {
		t.Fatalf("InsertIfAbsent: %v", err)
	}
	// Second insert must NOT overwrite.
	if err := repo.InsertIfAbsent(ctx, "k", "second"); err != nil {
		t.Fatalf("InsertIfAbsent (2nd): %v", err)
	}

	v, found, err := repo.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || v != "first" {
		t.Fatalf("got (%q, %v), want (\"first\", true)", v, found)
	}
}
```

- [ ] **Step 2: Run the test, expect failure to compile**

Run:

```bash
TEST_DATABASE_URL=postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable \
  go test -tags integration -run TestSettingsRepo ./internal/infrastructure/postgres/ -v
```

Expected: build failure — `undefined: postgres.NewSettingsRepo`.

- [ ] **Step 3: Write the sqlc query file**

`internal/infrastructure/postgres/sqlc/queries/sql/settings.sql`:

```sql
-- name: GetSetting :one
SELECT value FROM app_settings WHERE key = $1;

-- name: InsertSettingIfAbsent :exec
INSERT INTO app_settings (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO NOTHING;
```

- [ ] **Step 4: Generate sqlc code**

Run:

```bash
make sqlc
```

(If `sqlc` is not installed: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`, then re-run.)
Expected: regenerates `internal/infrastructure/postgres/sqlc/queries/settings.sql.go`, `models.go` (adds an `AppSetting` model), and `querier.go` (adds `GetSetting`, `InsertSettingIfAbsent`). No errors.

- [ ] **Step 5: Write the repo wrapper**

`internal/infrastructure/postgres/settings_repo.go`:

```go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

// SettingsRepo is a generic key/value store for runtime-provisioned config.
type SettingsRepo struct{ q *queries.Queries }

func NewSettingsRepo(pool *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{q: queries.New(pool)}
}

// Get returns the value for key. found is false (with nil error) when the key
// is absent.
func (r *SettingsRepo) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := r.q.GetSetting(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// InsertIfAbsent writes key=value only if key does not already exist. A losing
// writer in a concurrent race is a no-op, not an error.
func (r *SettingsRepo) InsertIfAbsent(ctx context.Context, key, value string) error {
	return r.q.InsertSettingIfAbsent(ctx, queries.InsertSettingIfAbsentParams{
		Key:   key,
		Value: value,
	})
}
```

- [ ] **Step 6: Run the test, expect pass**

Run:

```bash
TEST_DATABASE_URL=postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable \
  go test -tags integration -run TestSettingsRepo ./internal/infrastructure/postgres/ -v
```

Expected: PASS for both `TestSettingsRepo_GetMissingReturnsNotFound` and `TestSettingsRepo_InsertIfAbsentThenGet`.

- [ ] **Step 7: Commit**

```bash
git add internal/infrastructure/postgres/sqlc/queries/sql/settings.sql internal/infrastructure/postgres/sqlc/queries/ internal/infrastructure/postgres/settings_repo.go internal/infrastructure/postgres/settings_repo_integration_test.go
git commit -m "feat(deploy): add SettingsRepo over app_settings"
```

---

## Task 3: VAPID resolver (`internal/infrastructure/vapid`)

**Files:**
- Create: `internal/infrastructure/vapid/resolve.go`
- Test: `internal/infrastructure/vapid/resolve_test.go`

- [ ] **Step 1: Write the failing unit test**

`internal/infrastructure/vapid/resolve_test.go`:

```go
package vapid

import (
	"context"
	"testing"
)

// fakeStore is an in-memory Store for unit testing Resolve without a DB.
type fakeStore struct{ m map[string]string }

func newFakeStore() *fakeStore { return &fakeStore{m: map[string]string{}} }

func (s *fakeStore) Get(_ context.Context, key string) (string, bool, error) {
	v, ok := s.m[key]
	return v, ok, nil
}

func (s *fakeStore) InsertIfAbsent(_ context.Context, key, value string) error {
	if _, ok := s.m[key]; !ok {
		s.m[key] = value
	}
	return nil
}

func emptyEnv(string) string { return "" }

func TestResolve_GeneratesAndPersistsWhenEmpty(t *testing.T) {
	// Deterministic key generation for the test.
	orig := generateKeys
	generateKeys = func() (string, string, error) { return "PRIV", "PUB", nil }
	defer func() { generateKeys = orig }()

	store := newFakeStore()
	keys, source, err := Resolve(context.Background(), store, emptyEnv)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if source != "generated" {
		t.Fatalf("source = %q, want \"generated\"", source)
	}
	if keys.Public != "PUB" || keys.Private != "PRIV" {
		t.Fatalf("keys = %+v, want PUB/PRIV", keys)
	}
	if keys.Email != defaultEmail {
		t.Fatalf("email = %q, want %q", keys.Email, defaultEmail)
	}

	// Second call returns the SAME keys from the store, source "db".
	keys2, source2, err := Resolve(context.Background(), store, emptyEnv)
	if err != nil {
		t.Fatalf("Resolve (2nd): %v", err)
	}
	if source2 != "db" {
		t.Fatalf("source2 = %q, want \"db\"", source2)
	}
	if keys2 != keys {
		t.Fatalf("keys changed across calls: %+v vs %+v", keys2, keys)
	}
}

func TestResolve_AdoptsEnvThenDBOnly(t *testing.T) {
	store := newFakeStore()
	env := func(k string) string {
		switch k {
		case "VAPID_PUBLIC_KEY":
			return "ENVPUB"
		case "VAPID_PRIVATE_KEY":
			return "ENVPRIV"
		case "VAPID_EMAIL":
			return "mailto:env@example.com"
		}
		return ""
	}

	keys, source, err := Resolve(context.Background(), store, env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if source != "env" {
		t.Fatalf("source = %q, want \"env\"", source)
	}
	if keys.Public != "ENVPUB" || keys.Private != "ENVPRIV" || keys.Email != "mailto:env@example.com" {
		t.Fatalf("keys = %+v, want env values", keys)
	}

	// With env now cleared, the same keys still come from the DB.
	keys2, source2, err := Resolve(context.Background(), store, emptyEnv)
	if err != nil {
		t.Fatalf("Resolve (2nd): %v", err)
	}
	if source2 != "db" {
		t.Fatalf("source2 = %q, want \"db\"", source2)
	}
	if keys2 != keys {
		t.Fatalf("keys changed: %+v vs %+v", keys2, keys)
	}
}

func TestResolve_DBWinsOverEnv(t *testing.T) {
	store := newFakeStore()
	store.m["vapid_public"] = "DBPUB"
	store.m["vapid_private"] = "DBPRIV"
	store.m["vapid_email"] = "mailto:db@example.com"

	env := func(k string) string {
		if k == "VAPID_PUBLIC_KEY" {
			return "ENVPUB"
		}
		if k == "VAPID_PRIVATE_KEY" {
			return "ENVPRIV"
		}
		return ""
	}

	keys, source, err := Resolve(context.Background(), store, env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if source != "db" {
		t.Fatalf("source = %q, want \"db\"", source)
	}
	if keys.Public != "DBPUB" || keys.Private != "DBPRIV" {
		t.Fatalf("keys = %+v, want DB values", keys)
	}
}
```

- [ ] **Step 2: Run the test, expect failure to compile**

Run:

```bash
go test ./internal/infrastructure/vapid/ -v
```

Expected: build failure — `undefined: generateKeys`, `undefined: Resolve`, `undefined: defaultEmail`.

- [ ] **Step 3: Write the resolver**

`internal/infrastructure/vapid/resolve.go`:

```go
// Package vapid resolves the server's Web Push VAPID keypair.
//
// The database (via Store) is the single runtime source of truth. Env vars are
// only a one-time bootstrap seed: if the DB has no keys but VAPID_* env vars are
// set, they are adopted into the DB once (preserving existing push subscribers,
// whose subscriptions are bound to the public key) and never read again.
package vapid

import (
	"context"
	"fmt"

	webpushlib "github.com/SherClockHolmes/webpush-go"
)

const (
	keyPublic  = "vapid_public"
	keyPrivate = "vapid_private"
	keyEmail   = "vapid_email"

	// defaultEmail is used when no contact address is supplied. Push services
	// accept it; operators can override by writing the vapid_email setting.
	defaultEmail = "mailto:admin@example.com"
)

// generateKeys is the keypair generator, overridable in tests. The underlying
// function returns (privateKey, publicKey, error) in that order.
var generateKeys = webpushlib.GenerateVAPIDKeys

// Keys is a resolved VAPID keypair plus the contact email.
type Keys struct {
	Public  string
	Private string
	Email   string
}

// Store is the minimal persistence the resolver needs. *postgres.SettingsRepo
// satisfies it.
type Store interface {
	Get(ctx context.Context, key string) (string, bool, error)
	InsertIfAbsent(ctx context.Context, key, value string) error
}

// Resolve returns the VAPID keys, provisioning and persisting them if needed.
// source is "db", "env", or "generated" for logging.
func Resolve(ctx context.Context, store Store, getenv func(string) string) (Keys, string, error) {
	// 1. Steady state: keys already in the DB.
	pub, pubOK, err := store.Get(ctx, keyPublic)
	if err != nil {
		return Keys{}, "", fmt.Errorf("read vapid_public: %w", err)
	}
	priv, privOK, err := store.Get(ctx, keyPrivate)
	if err != nil {
		return Keys{}, "", fmt.Errorf("read vapid_private: %w", err)
	}
	if pubOK && privOK {
		return Keys{Public: pub, Private: priv, Email: emailOrDefault(ctx, store)}, "db", nil
	}

	// 2/3. Provision: adopt env keys if present, else generate fresh.
	var newPub, newPriv, source string
	if ep, epriv := getenv("VAPID_PUBLIC_KEY"), getenv("VAPID_PRIVATE_KEY"); ep != "" && epriv != "" {
		newPub, newPriv, source = ep, epriv, "env"
	} else {
		gp, gpub, gerr := generateKeys()
		if gerr != nil {
			return Keys{}, "", fmt.Errorf("generate vapid keys: %w", gerr)
		}
		newPub, newPriv, source = gpub, gp, "generated"
	}

	email := getenv("VAPID_EMAIL")
	if email == "" {
		email = defaultEmail
	}

	// Persist race-safely: a concurrent web container may win the insert.
	if err := store.InsertIfAbsent(ctx, keyPublic, newPub); err != nil {
		return Keys{}, "", fmt.Errorf("persist vapid_public: %w", err)
	}
	if err := store.InsertIfAbsent(ctx, keyPrivate, newPriv); err != nil {
		return Keys{}, "", fmt.Errorf("persist vapid_private: %w", err)
	}
	if err := store.InsertIfAbsent(ctx, keyEmail, email); err != nil {
		return Keys{}, "", fmt.Errorf("persist vapid_email: %w", err)
	}

	// Re-read so a race loser returns the winner's committed values.
	finalPub, _, err := store.Get(ctx, keyPublic)
	if err != nil {
		return Keys{}, "", fmt.Errorf("reread vapid_public: %w", err)
	}
	finalPriv, _, err := store.Get(ctx, keyPrivate)
	if err != nil {
		return Keys{}, "", fmt.Errorf("reread vapid_private: %w", err)
	}
	return Keys{Public: finalPub, Private: finalPriv, Email: emailOrDefault(ctx, store)}, source, nil
}

func emailOrDefault(ctx context.Context, store Store) string {
	email, ok, err := store.Get(ctx, keyEmail)
	if err != nil || !ok || email == "" {
		return defaultEmail
	}
	return email
}
```

- [ ] **Step 4: Run the test, expect pass**

Run:

```bash
go test ./internal/infrastructure/vapid/ -v
```

Expected: PASS for all three tests.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/vapid/
git commit -m "feat(deploy): VAPID resolver with DB-first, env-adopt, generate fallback"
```

---

## Task 4: Wire into `main.go`

**Files:**
- Modify: `main.go` (imports; pool/notifier/vapid-handler wiring)

- [ ] **Step 1: Add the `context` and `vapid` imports**

In `main.go`, change the import block. Replace:

```go
import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/boundaries/handler"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/infrastructure/webpush"
	"github.com/z3spinner/go-stop/internal/usecase"
	"github.com/z3spinner/go-stop/internal/version"
)
```

with:

```go
import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/boundaries/handler"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/infrastructure/vapid"
	"github.com/z3spinner/go-stop/internal/infrastructure/webpush"
	"github.com/z3spinner/go-stop/internal/usecase"
	"github.com/z3spinner/go-stop/internal/version"
)
```

- [ ] **Step 2: Resolve the VAPID keys and build the notifier from them**

In `main.go`, replace:

```go
	notifier := webpush.New(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_EMAIL"),
	)
```

with:

```go
	vapidKeys, vapidSource, err := vapid.Resolve(context.Background(), postgres.NewSettingsRepo(pool), os.Getenv)
	if err != nil {
		log.Fatalf("vapid: %v", err)
	}
	log.Printf("vapid: keys ready (source: %s)", vapidSource)

	notifier := webpush.New(vapidKeys.Public, vapidKeys.Private, vapidKeys.Email)
```

- [ ] **Step 3: Build the vapid handler from the resolved public key**

In `main.go`, replace:

```go
	vapidH := handler.NewVapidHandler(os.Getenv("VAPID_PUBLIC_KEY"))
```

with:

```go
	vapidH := handler.NewVapidHandler(vapidKeys.Public)
```

- [ ] **Step 4: Build and vet**

Run:

```bash
go build ./... && go vet ./...
```

Expected: no output (success). In particular, no "os imported and not used" — `os.Getenv` is still used elsewhere in `main.go`.

- [ ] **Step 5: Run the existing main serve test + unit tests**

Run:

```bash
go test ./... 2>&1 | tail -20
```

Expected: PASS (non-integration tests). `main_serve_test.go` still passes.

- [ ] **Step 6: Commit**

```bash
git add main.go
git commit -m "feat(deploy): resolve VAPID keys from DB on boot"
```

---

## Task 5: Strip required fields from `scalingo.json` + docs

**Files:**
- Modify: `scalingo.json`
- Modify: `README.md`
- Modify: `.env.example`

- [ ] **Step 1: Remove the three VAPID env entries from `scalingo.json`**

In `scalingo.json`, delete this block from `env` (the three VAPID entries), leaving `SITE_NAME` as the first entry:

```json
    "VAPID_PUBLIC_KEY": {
      "description": "VAPID public key for Web Push notifications (generate with: go run ./cmd/genvapid)",
      "required": true
    },
    "VAPID_PRIVATE_KEY": {
      "description": "VAPID private key for Web Push notifications",
      "required": true
    },
    "VAPID_EMAIL": {
      "description": "Contact email for Web Push — prefix with mailto: (e.g. mailto:you@example.com)",
      "required": true
    },
```

So `"env"` now opens directly with `"SITE_NAME"`. Leave all other entries unchanged.

- [ ] **Step 2: Validate the JSON**

Run:

```bash
python3 -m json.tool scalingo.json > /dev/null && echo "valid json"
```

Expected: `valid json`. Confirm no remaining `"required": true` entries:

```bash
grep -c '"required": true' scalingo.json
```

Expected: `0`.

- [ ] **Step 3: Update `.env.example`**

Replace the VAPID section of `.env.example`:

```
# Generate VAPID keys: go run github.com/SherClockHolmes/webpush-go/cmd/vapid-keygen@latest
VAPID_PUBLIC_KEY=
VAPID_PRIVATE_KEY=
VAPID_EMAIL=mailto:dev@localhost
```

with:

```
# VAPID keys for Web Push. Optional — if left blank, the server generates a
# keypair on first boot and stores it in the database (the same keys persist
# across restarts). Set them explicitly only if you want fixed local keys.
# Generate manually with: go run github.com/SherClockHolmes/webpush-go/cmd/vapid-keygen@latest
VAPID_PUBLIC_KEY=
VAPID_PRIVATE_KEY=
VAPID_EMAIL=mailto:dev@localhost
```

- [ ] **Step 4: Update `README.md` (Docker section)**

Replace:

```
```bash
# Generate VAPID keys (one-time)
go run github.com/SherClockHolmes/webpush-go/cmd/vapid-keygen@latest

cp .env.example .env
# Edit .env — set your VAPID keys and optionally SITE_NAME for your community

docker compose up --build
```
```

with:

```
```bash
cp .env.example .env
# Optionally edit .env to set SITE_NAME for your community.
# VAPID keys are generated automatically on first boot if left blank.

docker compose up --build
```
```

- [ ] **Step 5: Update `README.md` (manual setup section)**

Replace:

```
export VAPID_PUBLIC_KEY="your-public-key"
export VAPID_PRIVATE_KEY="your-private-key"
export VAPID_EMAIL="mailto:you@example.com"
export PORT=8080
```

with:

```
# VAPID keys are optional — generated and stored in the DB on first boot if unset.
export PORT=8080
```

- [ ] **Step 6: Commit**

```bash
git add scalingo.json README.md .env.example
git commit -m "feat(deploy): drop required VAPID env from scalingo.json; document auto-gen"
```

---

## Task 6: Full verification

- [ ] **Step 1: Run the full integration suite**

Run:

```bash
make test 2>&1 | tail -30
```

Expected: all packages PASS, including the new `TestSettingsRepo_*` tests.

- [ ] **Step 2: Verify a clean first-boot generates keys (manual smoke test)**

With a fresh dev DB (no VAPID env vars set), start the app and confirm the log line, then confirm the keys landed in the DB:

```bash
docker compose up --build   # watch for: vapid: keys ready (source: generated)
# in another shell:
psql "postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" \
  -c "SELECT key FROM app_settings ORDER BY key;"
```

Expected: log shows `vapid: keys ready (source: generated)`; the query returns `vapid_email`, `vapid_private`, `vapid_public`. Restarting the app logs `source: db` (same keys reused).

- [ ] **Step 3: Final commit (if any doc/log tweaks were needed during smoke test)**

```bash
git add -A
git commit -m "chore(deploy): verification pass for one-click deploy"
```

---

## Self-Review Notes

- **Spec coverage:** migration (Task 1), settings repo (Task 2), resolver with the 3-path resolution order incl. env-adopt + race-safe persist + re-read (Task 3), `main.go` wiring for both notifier and `/vapid` handler (Task 4), `scalingo.json` + docs incl. dangling `cmd/genvapid` removal (Task 5, the reference is deleted with the VAPID block), full + first-boot verification (Task 6). The in-site button already exists in `aboutBody` — noted, no task. PHONE_ENCRYPTION_KEY left untouched per spec.
- **Type consistency:** `Resolve(ctx, Store, func(string) string) (Keys, string, error)`, `Keys{Public,Private,Email}`, `Store{Get,InsertIfAbsent}`, `postgres.NewSettingsRepo`, `SettingsRepo.Get/InsertIfAbsent`, and sqlc `GetSetting`/`InsertSettingIfAbsentParams{Key,Value}` are used identically across tasks. `generateKeys`/`webpushlib.GenerateVAPIDKeys` return order is (private, public) — handled in Task 3 Step 3.
