// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// engagementThreshold is the minimum number of available rides required to skip
// the monthly motivational push notification.
const engagementThreshold = 10

// engagementSettingPrefix is prepended to "YYYY-MM" to form the idempotency key.
const engagementSettingPrefix = "engagement_reminder:"

// engagementTitles and engagementBodies hold the localised push text. French is
// the app's primary language and the fallback (via pick) when the subscriber's
// language cannot be determined.
var engagementTitles = map[string]string{
	"fr": "Proposez un trajet ce mois-ci ! 🚗",
	"en": "Share a ride this month! 🚗",
	"de": "Biete diese Monat eine Mitfahrt an! 🚗",
	"es": "¡Comparte un viaje este mes! 🚗",
	"it": "Condividi un viaggio questo mese! 🚗",
	"nl": "Deel een rit deze maand! 🚗",
}

var engagementBodies = map[string]string{
	"fr": "Le tableau est presque vide — soyez le premier à poster un trajet et aidez vos voisins à se déplacer.",
	"en": "The board is almost empty — be the first to post a ride and help your neighbours get moving.",
	"de": "Das Brett ist fast leer — sei der Erste, der eine Fahrt anbietet, und hilf deinen Nachbarn.",
	"es": "El tablón está casi vacío — sé el primero en publicar un trayecto y ayuda a tus vecinos a moverse.",
	"it": "La bacheca è quasi vuota — sii il primo a pubblicare un viaggio e aiuta i tuoi vicini a spostarsi.",
	"nl": "Het bord is bijna leeg — wees de eerste die een rit plaatst en help je buren op weg.",
}

// SendEngagementReminder is a monthly scheduled job that sends a motivational
// push notification to all subscribers when available rides fall below the
// threshold. It is idempotent: once the notification is sent for a given month,
// subsequent runs within that month are a no-op.
type SendEngagementReminder struct {
	rides    repository.RideRepository
	subs     repository.SubscriptionRepository
	settings repository.SettingsRepository
	notifier notification.Notifier
	loc      *time.Location
	// Clock is the time source; defaults to time.Now. Override in tests.
	Clock func() time.Time
}

func NewSendEngagementReminder(
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	settings repository.SettingsRepository,
	notifier notification.Notifier,
	loc *time.Location,
) *SendEngagementReminder {
	if loc == nil {
		loc = time.UTC
	}
	return &SendEngagementReminder{
		rides:    rides,
		subs:     subs,
		settings: settings,
		notifier: notifier,
		loc:      loc,
		Clock:    time.Now,
	}
}

// Execute checks whether it is the beginning of a new month and, if so, sends
// a motivational push notification to all subscribers when fewer than
// engagementThreshold rides are currently available. The run is skipped when the
// notification has already been sent for the current month.
func (uc *SendEngagementReminder) Execute() error {
	now := uc.Clock().In(uc.loc)
	log.Printf("engagement reminder: running (day=%d month=%s)", now.Day(), now.Format("2006-01"))

	// Only act during the first 3 days of the month so the job fires even when
	// the server restarts after day 1.
	if now.Day() > 3 {
		log.Printf("engagement reminder: skipped (day %d is outside window)", now.Day())
		return nil
	}

	monthKey := engagementSettingPrefix + now.Format("2006-01")
	ctx := context.Background()

	_, alreadySent, err := uc.settings.Get(ctx, monthKey)
	if err != nil {
		return fmt.Errorf("engagement reminder: check idempotency key: %w", err)
	}
	if alreadySent {
		log.Printf("engagement reminder: already sent for %s, skipping", now.Format("2006-01"))
		return nil
	}

	rides, err := uc.rides.FindAll()
	if err != nil {
		return fmt.Errorf("engagement reminder: count rides: %w", err)
	}
	count := len(rides)
	log.Printf("engagement reminder: available rides=%d threshold=%d", count, engagementThreshold)

	if count >= engagementThreshold {
		log.Printf("engagement reminder: enough rides (%d >= %d), no push sent", count, engagementThreshold)
		return nil
	}

	subs, err := uc.subs.FindAll()
	if err != nil {
		return fmt.Errorf("engagement reminder: fetch subscribers: %w", err)
	}
	if len(subs) == 0 {
		log.Printf("engagement reminder: no subscribers, skipping push")
		// Still mark as sent so we don't log repeatedly.
		return uc.settings.InsertIfAbsent(ctx, monthKey, "sent")
	}

	// Subscriptions carry no language preference, so fall back to French —
	// the app's primary language. pick() returns the French value when the
	// key is absent or empty, so this is safe for all future callers too.
	msg := domain.Message{
		Title: pick(engagementTitles, "fr"),
		Body:  pick(engagementBodies, "fr"),
		URL:   "/",
	}

	sent := 0
	for _, sub := range subs {
		if err := uc.notifier.Send(sub, msg); err != nil {
			log.Printf("engagement reminder: push error endpoint=%.20s...: %v", sub.Endpoint, err)
			if strings.Contains(err.Error(), "410") {
				_ = uc.subs.DeleteByEndpoint(sub.Endpoint)
			}
		} else {
			sent++
		}
	}
	log.Printf("engagement reminder: push sent to %d/%d subscribers", sent, len(subs))

	// Record that we have sent the reminder for this month. InsertIfAbsent is
	// race-safe: a concurrent call (unlikely in single-instance but possible)
	// that reaches this point will simply lose the race and be a no-op.
	return uc.settings.InsertIfAbsent(ctx, monthKey, "sent")
}
