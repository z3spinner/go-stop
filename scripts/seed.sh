#!/usr/bin/env bash
#
# Seed the dev database with realistic rides and search alerts.
#
# Posts through the running app's HTTP API (not raw SQL) so that expiry,
# location normalisation and validation match production exactly and the seed
# survives schema changes. Handy after `make test`, which truncates the dev DB.
#
# Usage:
#   ./scripts/seed.sh                 # against http://localhost:8080
#   BASE_URL=http://localhost:3000 ./scripts/seed.sh
#
# The seed is additive — re-running adds more rows rather than replacing.

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API="$BASE_URL/api"

# RFC3339 timestamp a relative amount from now, e.g. rfc3339 "+2 hours".
rfc3339() { date -u -d "$1" +%Y-%m-%dT%H:%M:%SZ; }
# YYYY-MM-DD date a relative amount from now, e.g. ymd "+2 days".
ymd() { date -u -d "$1" +%F; }

# post <path> <json> <label> — POST and report the resulting status.
post() {
  local path="$1" body="$2" label="$3" code
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/$path" \
    -H 'Content-Type: application/json' -d "$body")
  if [[ "$code" == "201" ]]; then
    echo "  ✓ $label"
  else
    echo "  ✗ $label (HTTP $code)" >&2
  fi
}

ride() { # driver phone origin destination departure_at flexibility
  post rides "$(printf '{"driver_name":"%s","phone":"%s","origin":"%s","destination":"%s","departure_at":"%s","flexibility":%s}' \
    "$1" "$2" "$3" "$4" "$5" "$6")" "ride  $3 → $4"
}

# request_at: a search alert for a specific date+time (RFC3339).
request_at() { # searcher phone origin destination departure_at flexibility
  post requests "$(printf '{"searcher_name":"%s","phone":"%s","origin":"%s","destination":"%s","departure_at":"%s","flexibility":%s}' \
    "$1" "$2" "$3" "$4" "$5" "$6")" "alert $3 → $4 (le $5)"
}
# request_day: a search alert for a whole day (YYYY-MM-DD).
request_day() { # searcher phone origin destination departure_date flexibility
  post requests "$(printf '{"searcher_name":"%s","phone":"%s","origin":"%s","destination":"%s","departure_date":"%s","flexibility":%s}' \
    "$1" "$2" "$3" "$4" "$5" "$6")" "alert $3 → $4 (jour $5)"
}
# request_daily: a recurring daily alert at a time of day (HH:MM).
request_daily() { # searcher phone origin destination departure_time flexibility
  post requests "$(printf '{"searcher_name":"%s","phone":"%s","origin":"%s","destination":"%s","departure_time":"%s","flexibility":%s}' \
    "$1" "$2" "$3" "$4" "$5" "$6")" "alert $3 → $4 (tous les jours à $5)"
}

echo "Waiting for $BASE_URL …"
curl -s -m 30 --retry 30 --retry-delay 1 --retry-all-errors -o /dev/null "$API/config" \
  || { echo "App not reachable at $BASE_URL — start it (docker compose up) first." >&2; exit 1; }

echo "Seeding rides (drivers offering seats) …"
ride "Camille"  "0612340001" "Saillans" "Crest"      "$(rfc3339 '+2 hours')"        15
ride "Lucas"    "0612340002" "Crest"    "Saillans"   "$(rfc3339 '+4 hours')"        15
ride "Naïma"    "0612340003" "Saillans" "Die"        "$(rfc3339 '+6 hours')"        30
ride "Thomas"   "0612340004" "Saillans" "Valence"    "$(rfc3339 'tomorrow 08:00')"  20
ride "Élodie"   "0612340005" "Die"      "Saillans"   "$(rfc3339 'tomorrow 18:30')"  15
ride "Mehdi"    "0612340006" "Saillans" "Grenoble"   "$(rfc3339 '+2 days 07:30')"   45

echo "Seeding search alerts (riders looking) …"
request_at    "Sophie"  "0698760001" "Saillans"   "Crest"    "$(rfc3339 'tomorrow 08:30')" 20
request_day   "Antoine" "0698760002" "Saillans"   "Valence"  "$(ymd '+2 days')"            30
request_daily "Inès"    "0698760003" "Crest"      "Saillans" "17:30"                       15
request_at    "Yann"    "0698760004" "Montélimar" "Saillans" "$(rfc3339 '+3 days 19:00')"  30
# anytime alert — no date or time, matches any matching ride
post requests '{"searcher_name":"Clara","phone":"0698760005","origin":"Saillans","destination":"Die","flexibility":0}' \
  "alert Saillans → Die (n'importe quand)"

echo "Done."
