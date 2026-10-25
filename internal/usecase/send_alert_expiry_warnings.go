// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"fmt"
	"log"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// SendAlertExpiryWarnings asks the owner of each open-ended alert (anytime or
// daily) that expires within domain.AlertExpiryNotice whether to keep it: one
// push and inbox item per expiry, linking to My searches where "Prolonger"
// renews it (RenewRequest). An alert nobody renews simply expires, so alerts no
// longer outlive the people who posted them. Nothing is sent during quiet hours;
// due warnings go out at the end of them. Push-only text is French, like the
// other server-initiated pushes; the inbox renders the kind in the reader's
// language.
type SendAlertExpiryWarnings struct {
	alerts   repository.AlertExpiryRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
	quiet    QuietHours
	tz       *time.Location
	now      func() time.Time
}

func NewSendAlertExpiryWarnings(
	alerts repository.AlertExpiryRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
	quiet QuietHours,
	tz *time.Location,
) *SendAlertExpiryWarnings {
	if tz == nil {
		tz = time.UTC
	}
	return &SendAlertExpiryWarnings{alerts: alerts, subs: subs, notifier: notifier, quiet: quiet, tz: tz, now: time.Now}
}

func (uc *SendAlertExpiryWarnings) Execute() error {
	now := uc.now()
	if uc.quiet.ContainsTime(now, uc.tz) {
		return nil
	}
	due, err := uc.alerts.FindExpiringBefore(now.Add(domain.AlertExpiryNotice))
	if err != nil {
		return err
	}
	for _, req := range due {
		sendToAll(req.Phone, domain.Message{
			Title:       "Votre alerte expire bientôt",
			Body:        fmt.Sprintf("%s → %s : gardez-la en un clic", req.Origin, req.Destination),
			URL:         "/my-searches?alert=" + req.ID,
			Origin:      req.Origin,
			Destination: req.Destination,
			Kind:        domain.KindAlertExpiring,
			DedupKey:    domain.AlertExpiryKey(req.ID, req.ExpiresAt),
			RequestID:   req.ID,
		}, uc.subs, uc.notifier)
		if err := uc.alerts.MarkExpiryWarned(req.ID, req.ExpiresAt); err != nil {
			log.Printf("alert expiry warning: mark warned: %v", err)
		}
	}
	return nil
}
