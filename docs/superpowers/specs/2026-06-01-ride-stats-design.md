# Ride Stats & Feedback — Design Spec

## Goal

Track how many rides were actually shared between locations by collecting post-ride feedback from drivers, and surface a "top routes this week" leaderboard on the home page.

---

## Data Model

### New table: `ride_stats`

Persists after rides are deleted (rides expire daily, stats must survive).

```sql
CREATE TABLE ride_stats (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    origin       VARCHAR(200) NOT NULL,
    destination  VARCHAR(200) NOT NULL,
    ride_date    DATE         NOT NULL,
    taken        BOOLEAN      NOT NULL,   -- true = someone came along
    recorded_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ride_stats_route ON ride_stats(origin, destination);
CREATE INDEX idx_ride_stats_date  ON ride_stats(recorded_at);
```

### New column on `rides`

```sql
ALTER TABLE rides ADD COLUMN feedback_given BOOLEAN NOT NULL DEFAULT FALSE;
```

`feedback_given = true` means this ride has been answered (prevents duplicate push prompts and hides the inline prompt).

---

## API

### `POST /api/rides/:id/feedback`

**Auth:** phone number in request body (same lightweight auth as delete).

**Request body:**
```json
{ "phone": "06 12 34 56 78", "taken": true }
```

**Logic:**
1. `FindByID(id)` — 404 if not found
2. Check `ride.Phone == phone` — 403 if mismatch
3. Insert into `ride_stats(origin, destination, ride_date, taken)`
4. Set `rides.feedback_given = true`
5. Return 204 No Content

**Errors:**
- 400 if body is malformed
- 403 if phone doesn't match
- 404 if ride not found or already expired

### `GET /api/stats`

No auth required.

**Response:**
```json
{
  "top_routes": [
    { "origin": "Saillans", "destination": "Crest", "count": 4 },
    { "origin": "Saillans", "destination": "Die",   "count": 3 }
  ],
  "total_confirmed": 42,
  "total_this_week": 9
}
```

**Query — top routes:**
```sql
SELECT origin, destination, COUNT(*) AS count
FROM ride_stats
WHERE taken = true
  AND recorded_at >= DATE_TRUNC('week', NOW())
GROUP BY origin, destination
ORDER BY count DESC
LIMIT 5;
```

**Query — totals:**
```sql
SELECT
  COUNT(*) FILTER (WHERE taken = true) AS total_confirmed,
  COUNT(*) FILTER (WHERE taken = true AND recorded_at >= DATE_TRUNC('week', NOW())) AS total_this_week
FROM ride_stats;
```

---

## Backend — New Components

### Domain
`internal/domain/stat.go`
```go
type RouteStat struct {
    Origin      string
    Destination string
    Count       int
}

type Stats struct {
    TopRoutes      []RouteStat
    TotalConfirmed int
    TotalThisWeek  int
}
```

### Repository interface
`internal/boundaries/repository/stat_repository.go`
```go
type StatRepository interface {
    Save(origin, destination string, rideDate time.Time, taken bool) error
    GetStats() (domain.Stats, error)
}
```

### Use cases
- `RecordFeedback` — finds ride, phone-checks, saves stat, marks `feedback_given`
- `GetStats` — delegates to `StatRepository.GetStats()`

### Handlers
- `FeedbackHandler.Post(c *gin.Context)` — handles `POST /api/rides/:id/feedback`
- `StatsHandler.Get(c *gin.Context)` — handles `GET /api/stats`

### Feedback push notification (in existing expiry cron)

The hourly goroutine in `main.go` gets a third task:

```
Find rides WHERE departure_at BETWEEN (NOW() - 23h) AND (NOW() - 30min)
  AND feedback_given = false
  AND expires_at > NOW()
For each:
  Look up subscription by ride.Phone
  If found: send push notification asking for feedback
```

Push payload:
```json
{
  "title": "Votre trajet est-il parti avec des passagers ?",
  "body": "Saillans → Crest · jeu. 15 juin",
  "url": "/my-rides"
}
```

English variant:
```json
{
  "title": "Did anyone join your ride?",
  "body": "Saillans → Crest · Thu 15 Jun",
  "url": "/my-rides"
}
```

Note: the server sends English/French notification text based on... it doesn't know the user's language. For simplicity, send French (the primary language of the target community). A future improvement could store language preference with the subscription.

### New use case: `SendFeedbackReminders`
Finds rides in the 30 min–23h post-departure window with `feedback_given = false`, looks up each driver's push subscription, sends the feedback request notification.

Called from the hourly cron alongside `ExpireRides` and `ExpireRequests`.

---

## Existing component changes

### `RideRepository` — two new methods
```go
FindPendingFeedback() ([]domain.Ride, error)   // departure 30min–23h ago, feedback_given=false, not expired
SetFeedbackGiven(id string) error
```

### Migration file
`db/migrations/002_add_stats.sql`

---

## Frontend

### i18n strings (both EN and FR)

| Key | EN | FR |
|---|---|---|
| `feedbackTitle` | Did anyone join your ride? | Quelqu'un est-il venu ? |
| `feedbackYes` | Yes, someone joined | Oui, quelqu'un est venu |
| `feedbackNo` | No, I drove alone | Non, j'ai conduit seul(e) |
| `feedbackThanks` | Thanks! | Merci ! |
| `statsTitle` | This week | Cette semaine |
| `statsEmpty` | No confirmed rides yet this week. | Aucun trajet confirmé cette semaine. |
| `statsAllTime` | All time: {n} confirmed rides | Depuis le début : {n} trajets confirmés |
| `btnAllStats` | See all stats | Voir toutes les stats |
| `statsPageTitle` | Stats | Statistiques |

### Home page — stats section

Shown below the two main buttons, only if `top_routes.length > 0`:

```
──────────────────────────────
Cette semaine
Saillans → Crest          4 ✓
Saillans → Die            3 ✓
Crest → Valence           2 ✓
            Voir toutes les stats →
──────────────────────────────
```

Loaded on `renderHome()` via `GET /api/stats`. If the call fails or returns empty, the section is silently omitted.

### Stats page (`/stats`)

Route: `pushRoute('/stats')`, restored by `handleDeepLink()`.

Displays:
- Headline: "42 trajets confirmés depuis le début"
- Subtitle: "9 cette semaine"
- Table: all routes sorted by this week's count, with all-time count alongside
- ← Back button

### "Mes trajets" — inline feedback

After loading the user's rides, any ride where `departure_at < NOW()` shows a feedback card:

```
┌────────────────────────────┐
│ Saillans → Crest           │
│ jeu. 15 juin à 08:30       │
│ Quelqu'un est-il venu ?    │
│ [Oui ✓]  [Non]             │
└────────────────────────────┘
```

On click: `POST /api/rides/:id/feedback`, card shows "Merci !" and collapses.

The `feedback_given` field is included in the `GET /api/rides?X-Phone: ...` response so the app knows which rides have already been answered.

---

## Migration strategy

The new migration `002_add_stats.sql` runs automatically via `docker-entrypoint-initdb.d` for fresh installs.

For existing deployments (Scalingo): run manually:
```bash
scalingo --app my-app run psql $DATABASE_URL < db/migrations/002_add_stats.sql
```

---

## What is NOT in scope

- Stats per driver
- Passenger counts (only yes/no)
- Historical weekly comparisons
- Admin dashboard
- Language-aware push notifications (always French for now)
