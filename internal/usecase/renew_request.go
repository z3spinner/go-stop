// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// RenewRequest keeps an open-ended alert (anytime or daily) for another
// domain.AlertLifetime from now. Only its owner may renew it, and a dated
// request can't be (it expires with its date). The expiry warning, if one was
// sent, counts as handled.
type RenewRequest struct {
	requests repository.RequestRepository
	alerts   repository.AlertExpiryRepository
	notifier notification.Notifier
	now      func() time.Time
}

func NewRenewRequest(requests repository.RequestRepository, alerts repository.AlertExpiryRepository, notifier notification.Notifier) *RenewRequest {
	return &RenewRequest{requests: requests, alerts: alerts, notifier: notifier, now: time.Now}
}

func (uc *RenewRequest) Execute(id, phone string) (domain.Request, error) {
	req, err := uc.requests.FindByID(id)
	if err != nil {
		return domain.Request{}, ErrNotFound
	}
	if phone == "" || req.Phone != phone {
		return domain.Request{}, ErrUnauthorized
	}
	if !isOpenEndedAlert(req) {
		return domain.Request{}, ErrNotRenewable
	}
	oldExpiry := req.ExpiresAt
	req.ExpiresAt = uc.now().Add(domain.AlertLifetime)
	if err := uc.alerts.SetExpiry(id, req.ExpiresAt); err != nil {
		return domain.Request{}, err
	}
	uc.notifier.Acknowledge(phone, domain.AlertExpiryKey(id, oldExpiry))
	return req, nil
}

// isOpenEndedAlert: an anytime alert (no date, no time) or a daily one (no date,
// a time on the 1970-01-01 sentinel day) — the alerts with no date of their own
// to expire with.
func isOpenEndedAlert(req domain.Request) bool {
	return req.Date.IsZero() && (req.DepartureAt.IsZero() || req.DepartureAt.Year() == 1970)
}
