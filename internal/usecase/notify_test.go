// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockNotifier struct {
	called   bool
	lastMsg  domain.Message
	err      error
	recorded []recordedMsg // every Record call (the inbox copy), in order
	acked    []string      // every Acknowledge call: "phone/key"
	sent     []domain.Message
}

type recordedMsg struct {
	phone string
	msg   domain.Message
}

func (m *mockNotifier) Send(sub domain.Subscription, msg domain.Message) error {
	m.called = true
	m.lastMsg = msg
	m.sent = append(m.sent, msg)
	return m.err
}

// Record returns a fake inbox id ("inbox-<n>") for messages with a Kind.
func (m *mockNotifier) Record(phone string, msg domain.Message) string {
	m.recorded = append(m.recorded, recordedMsg{phone: phone, msg: msg})
	if msg.Kind == "" {
		return ""
	}
	return fmt.Sprintf("inbox-%d", len(m.recorded))
}

func (m *mockNotifier) Acknowledge(phone string, keys ...string) {
	for _, k := range keys {
		m.acked = append(m.acked, phone+"/"+k)
	}
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
func (m *mockSubRepoNotify) FindAll() ([]domain.Subscription, error) { return m.subs, nil }
func (m *mockSubRepoNotify) Delete(string) error                     { return nil }
func (m *mockSubRepoNotify) DeleteByEndpoint(string) error           { return nil }

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
	usecase.NotifySearcher(domain.Request{ID: "req-1", Phone: "555-0001"}, ride, repo, counter)
	if counter.count != 2 {
		t.Errorf("expected 2 sends (one per device), got %d", counter.count)
	}
}

func TestNotifySearcher_NoSubscription_DoesNothing(t *testing.T) {
	n := &countingNotifier{}
	repo := &mockSubRepoNotify{} // empty
	usecase.NotifySearcher(domain.Request{ID: "req-1", Phone: "555-0001"}, domain.Ride{ID: "x"}, repo, n)
	if n.count != 0 {
		t.Errorf("expected 0 sends, got %d", n.count)
	}
}

func TestNotifySearcher_Removes410GoneSubscription(t *testing.T) {
	sub := domain.Subscription{Phone: "555-0001", Endpoint: "https://fcm.example/stale"}
	gone := &goneNotifier{}
	repo := &trackingSubRepo{subs: []domain.Subscription{sub}}
	usecase.NotifySearcher(domain.Request{ID: "req-1", Phone: "555-0001"}, domain.Ride{ID: "x"}, repo, gone)
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

func (n *countingNotifier) Record(string, domain.Message) string { return "" }
func (n *countingNotifier) Acknowledge(string, ...string)        {}

type goneNotifier struct{}

func (n *goneNotifier) Send(_ domain.Subscription, _ domain.Message) error {
	return errors.New("push service returned status 410")
}

func (n *goneNotifier) Record(string, domain.Message) string { return "" }
func (n *goneNotifier) Acknowledge(string, ...string)        {}

type trackingSubRepo struct {
	subs    []domain.Subscription
	deleted bool
}

func (r *trackingSubRepo) Save(domain.Subscription) error { return nil }
func (r *trackingSubRepo) FindByPhone(string) ([]domain.Subscription, error) {
	return r.subs, nil
}
func (r *trackingSubRepo) FindAll() ([]domain.Subscription, error) { return r.subs, nil }
func (r *trackingSubRepo) Delete(string) error                     { return nil }
func (r *trackingSubRepo) DeleteByEndpoint(string) error {
	r.deleted = true
	return nil
}
