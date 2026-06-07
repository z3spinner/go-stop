// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestSendFeedbackReminders_SendsPushToDriver(t *testing.T) {
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{
		{ID: "fq-1", RideID: "ride-1", Phone: "555-0001", Origin: "Saillans", Destination: "Crest",
			DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC)},
	}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(queue, subs, n, 2, 3)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected push notification to be sent")
	}
	if n.lastMsg.URL != "/rides/ride-1/feedback" {
		t.Errorf("expected URL /rides/ride-1/feedback, got %s", n.lastMsg.URL)
	}
	if len(queue.marked) != 1 || queue.marked[0] != "fq-1" {
		t.Error("expected task fq-1 marked sent")
	}
	if !queue.deleteExhaustedOK {
		t.Error("expected DeleteExhausted to be called")
	}
}

func TestSendFeedbackReminders_SkipsIfNoSubscription(t *testing.T) {
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{
		{ID: "fq-1", RideID: "ride-1", Phone: "555-no-sub"},
	}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(queue, subs, n, 2, 3)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send when driver has no subscription")
	}
}

func TestSendFeedbackReminders_NoDueTasks_NoNotifications(t *testing.T) {
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(queue, subs, n, 2, 3)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send any notification with no due tasks")
	}
}
