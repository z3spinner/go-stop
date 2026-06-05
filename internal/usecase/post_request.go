// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type PostRequest struct {
	requests repository.RequestRepository
	rides    repository.RideRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
}

func NewPostRequest(
	requests repository.RequestRepository,
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *PostRequest {
	return &PostRequest{requests: requests, rides: rides, subs: subs, notifier: notifier}
}

func (uc *PostRequest) Execute(req domain.Request) (domain.Request, error) {
	req.ID = uuid.New().String()
	req.PostedAt = time.Now()
	switch {
	case req.Date.IsZero() && req.DepartureAt.IsZero(): // anytime — no date/time constraint
		req.ExpiresAt = time.Now().AddDate(1, 0, 0)
	case req.Date.IsZero() && req.DepartureAt.Year() == 1970: // daily — recurring time-of-day (1970-01-01 sentinel)
		req.ExpiresAt = time.Now().AddDate(1, 0, 0)
	case !req.Date.IsZero() && req.DepartureAt.IsZero(): // day — a given date, any time
		req.ExpiresAt = req.Date.AddDate(0, 0, 1)
	default: // time — a concrete one-off date+time. The handler leaves the Date column
		// NULL (departure_at carries the day), so derive Date here; otherwise a
		// time alert is mistaken for a daily one and expires in a year instead of
		// the day after (matching rides).
		req.Date = time.Date(req.DepartureAt.Year(), req.DepartureAt.Month(), req.DepartureAt.Day(), 0, 0, 0, 0, req.DepartureAt.Location())
		req.ExpiresAt = req.Date.AddDate(0, 0, 1)
	}

	if err := uc.requests.Save(req); err != nil {
		return domain.Request{}, err
	}

	matching, err := uc.rides.FindMatching(req)
	if err != nil {
		return domain.Request{}, err
	}

	for _, ride := range matching {
		NotifyDriver(ride.Phone, req, uc.subs, uc.notifier)
	}
	return req, nil
}
