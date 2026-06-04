// Command backfill-matches notifies searchers about rides that became matchable
// only after the accent/case/whitespace-insensitive matching change (migration
// 010). Notifications are event-driven and fire once at post time, so existing
// ride/request pairs that newly match are never notified on their own.
//
// It considers ONLY genuinely new matches — pairs that match under the
// normalized rules but did NOT match under the old LOWER()-based rules (i.e.
// accent- or whitespace-only differences). Pairs that already matched
// case-insensitively were notified at post time and are skipped, so re-running
// the live matching logic here can't spam users about rides they already saw.
//
// Defaults to a dry run: it prints exactly who would be notified and sends
// nothing. Pass --send to actually deliver the push notifications.
//
//	scalingo --app go-stop-saillans run go run ./cmd/backfill-matches            # dry run
//	scalingo --app go-stop-saillans run go run ./cmd/backfill-matches --send     # for real
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/infrastructure/webpush"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// lastN returns the last n chars of s, for masking phone numbers in output.
func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// lowerEq replicates the OLD matching predicate exactly: case-folded equality
// with no trimming or accent handling. A pair that is lowerEq on both endpoints
// already matched before migration 010 and must NOT be re-notified.
func lowerEq(a, b string) bool { return strings.ToLower(a) == strings.ToLower(b) }

func main() {
	var send bool
	cmd := &cobra.Command{
		Use:   "backfill-matches",
		Short: "Notify searchers of rides that newly match after accent/whitespace normalization",
		RunE: func(*cobra.Command, []string) error {
			return run(send)
		},
	}
	cmd.Flags().BoolVar(&send, "send", false, "actually send notifications (default: dry run, sends nothing)")
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(send bool) error {
	pool, err := postgres.NewPool()
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	graceMins := 60
	if v := os.Getenv("RIDE_GRACE_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			graceMins = n
		}
	}
	rideRepo := postgres.NewRideRepo(pool, graceMins)
	requestRepo := postgres.NewRequestRepo(pool, graceMins)
	subRepo := postgres.NewSubscriptionRepo(pool)
	notifier := webpush.New(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_EMAIL"),
	)

	if send {
		fmt.Println("MODE: SEND — notifications WILL be delivered.")
	} else {
		fmt.Println("MODE: DRY RUN — no notifications sent. Re-run with --send to deliver.")
	}

	requests, err := requestRepo.FindAllActive()
	if err != nil {
		return fmt.Errorf("list active requests: %w", err)
	}

	sent := map[string]bool{} // dedup key: searcherPhone|rideID
	var newPairs, withSub, delivered int

	for _, req := range requests {
		rides, err := rideRepo.FindMatching(req)
		if err != nil {
			return fmt.Errorf("match rides for request %s: %w", req.ID, err)
		}
		for _, ride := range rides {
			// Skip anything that already matched under the old LOWER() rules.
			if lowerEq(req.Origin, ride.Origin) && lowerEq(req.Destination, ride.Destination) {
				continue
			}
			key := req.Phone + "|" + ride.ID
			if sent[key] {
				continue
			}
			sent[key] = true
			newPairs++

			subs, _ := subRepo.FindByPhone(req.Phone)
			hasSub := len(subs) > 0
			if hasSub {
				withSub++
			}

			fmt.Printf("  • search %q→%q (%s, ***%s) ↔ ride %q→%q (%s) — push:%s\n",
				req.Origin, req.Destination, req.SearcherName, lastN(req.Phone, 3),
				ride.Origin, ride.Destination, ride.DriverName,
				map[bool]string{true: "yes", false: "no-subscription"}[hasSub])

			if send && hasSub {
				usecase.NotifySearcher(req.Phone, ride, subRepo, notifier)
				delivered++
			}
		}
	}

	fmt.Printf("\nSummary: %d active requests scanned, %d newly-matchable pairs (%d with a push subscription).\n",
		len(requests), newPairs, withSub)
	if send {
		fmt.Printf("Delivered %d notifications.\n", delivered)
	} else {
		fmt.Println("Dry run — nothing sent. Re-run with --send to notify the subscribed searchers above.")
	}
	return nil
}
