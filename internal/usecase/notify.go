// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"fmt"
	"log"
	"strings"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// sendToAll delivers msg to every subscription for phone.
// Stale subscriptions (push service returns 410) are removed automatically.
func sendToAll(phone string, msg domain.Message, subs repository.SubscriptionRepository, notifier notification.Notifier) {
	subList, err := subs.FindByPhone(phone)
	if err != nil {
		return // no subscription — nothing to do
	}
	for _, sub := range subList {
		if err := notifier.Send(sub, msg); err != nil {
			log.Printf("push send error phone=***%s: %v", lastN(phone, 3), err)
			if strings.Contains(err.Error(), "410") {
				_ = subs.DeleteByEndpoint(sub.Endpoint)
			}
		} else {
			log.Printf("push sent ok phone=***%s", lastN(phone, 3))
		}
	}
}

// enqueueAndNotifyMatches pushes a "ride available" notification to each matching
// searcher not already notified for this ride. Enqueue reports whether the
// ride↔request pair was newly recorded; only new pairs are pushed, so re-running
// matching after a ride edit never re-pings an earlier match. On a fresh post
// every match is new, so behaviour is unchanged.
func enqueueAndNotifyMatches(ride domain.Ride, matches []domain.Request, queue repository.NotificationQueueRepository, subs repository.SubscriptionRepository, notifier notification.Notifier) {
	for _, req := range matches {
		inserted, _ := queue.Enqueue(ride.ID, req.ID, req.Phone)
		if !inserted {
			continue // already notified for this ride
		}
		NotifySearcher(req.Phone, ride, subs, notifier)
		_ = queue.MarkSentByRideAndRequest(ride.ID, req.ID)
	}
}

func NotifySearcher(phone string, ride domain.Ride, subs repository.SubscriptionRepository, notifier notification.Notifier) {
	msg := domain.Message{
		Title:       "Ride available!",
		Body:        fmt.Sprintf("%s is driving from %s to %s", ride.DriverName, ride.Origin, ride.Destination),
		URL:         "/rides/" + ride.ID,
		ContactName: ride.DriverName,
		Phone:       ride.Phone,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: ride.DepartureAt,
	}
	sendToAll(phone, msg, subs, notifier)
}

func NotifyDriver(phone string, req domain.Request, subs repository.SubscriptionRepository, notifier notification.Notifier) {
	msg := domain.Message{
		Title:       "Someone needs a ride!",
		Body:        fmt.Sprintf("%s needs a ride from %s to %s", req.SearcherName, req.Origin, req.Destination),
		URL:         "/my-rides",
		ContactName: req.SearcherName,
		Phone:       req.Phone,
		Origin:      req.Origin,
		Destination: req.Destination,
		DepartureAt: req.DepartureAt,
	}
	sendToAll(phone, msg, subs, notifier)
}
