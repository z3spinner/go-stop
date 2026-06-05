// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockNotifier struct {
	called  bool
	lastMsg domain.Message
	err     error
}

func (m *mockNotifier) Send(sub domain.Subscription, msg domain.Message) error {
	m.called = true
	m.lastMsg = msg
	return m.err
}

type mockSubRepoNotify struct {
	subs []domain.Subscription
}

func (m *mockSubRepoNotify) Save(domain.Subscription) error { return nil }
func (m *mockSubRepoNotify) FindByPhone(string) ([]domain.Subscription, error) {
	if len(m.subs) == 0 {
		return nil, errors.New("not found")
	}
	return m.subs, nil
}
func (m *mockSubRepoNotify) Delete(string) error           { return nil }
func (m *mockSubRepoNotify) DeleteByEndpoint(string) error { return nil }

func TestNotifySearcher_SendsToAllDevices(t *testing.T) {
	sub1 := domain.Subscription{Phone: "555-0001", Endpoint: "https://fcm.example/1"}
	sub2 := domain.Subscription{Phone: "555-0001", Endpoint: "https://fcm.example/2"}
	ride := domain.Ride{
		ID: "ride-1", DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Train Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	}
	counter := &countingNotifier{}
	repo := &mockSubRepoNotify{subs: []domain.Subscription{sub1, sub2}}
	usecase.NotifySearcher("555-0001", ride, repo, counter)
	if counter.count != 2 {
		t.Errorf("expected 2 sends (one per device), got %d", counter.count)
	}
}

func TestNotifySearcher_NoSubscription_DoesNothing(t *testing.T) {
	n := &countingNotifier{}
	repo := &mockSubRepoNotify{} // empty
	usecase.NotifySearcher("555-0001", domain.Ride{ID: "x"}, repo, n)
	if n.count != 0 {
		t.Errorf("expected 0 sends, got %d", n.count)
	}
}

func TestNotifySearcher_Removes410GoneSubscription(t *testing.T) {
	sub := domain.Subscription{Phone: "555-0001", Endpoint: "https://fcm.example/stale"}
	gone := &goneNotifier{}
	repo := &trackingSubRepo{subs: []domain.Subscription{sub}}
	usecase.NotifySearcher("555-0001", domain.Ride{ID: "x"}, repo, gone)
	if !repo.deleted {
		t.Error("expected stale subscription to be deleted after 410 Gone")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

type countingNotifier struct{ count int }

func (n *countingNotifier) Send(_ domain.Subscription, _ domain.Message) error {
	n.count++
	return nil
}

type goneNotifier struct{}

func (n *goneNotifier) Send(_ domain.Subscription, _ domain.Message) error {
	return errors.New("push service returned status 410")
}

type trackingSubRepo struct {
	subs    []domain.Subscription
	deleted bool
}

func (r *trackingSubRepo) Save(domain.Subscription) error { return nil }
func (r *trackingSubRepo) FindByPhone(string) ([]domain.Subscription, error) {
	return r.subs, nil
}
func (r *trackingSubRepo) Delete(string) error { return nil }
func (r *trackingSubRepo) DeleteByEndpoint(string) error {
	r.deleted = true
	return nil
}
