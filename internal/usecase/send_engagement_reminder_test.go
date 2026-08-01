// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// ── mock settings repository ──────────────────────────────────────────────────

type mockSettingsRepo struct {
	data         map[string]string
	insertCalled []string
	getErr       error
	insertErr    error
}

func newMockSettings() *mockSettingsRepo {
	return &mockSettingsRepo{data: map[string]string{}}
}

func (m *mockSettingsRepo) Get(_ context.Context, key string) (string, bool, error) {
	if m.getErr != nil {
		return "", false, m.getErr
	}
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *mockSettingsRepo) InsertIfAbsent(_ context.Context, key, value string) error {
	if m.insertErr != nil {
		return m.insertErr
	}
	m.insertCalled = append(m.insertCalled, key)
	if _, ok := m.data[key]; !ok {
		m.data[key] = value
	}
	return nil
}

// ── subscription mock with FindAll ───────────────────────────────────────────

type mockSubRepoAll struct {
	all        []domain.Subscription
	deleted    []string
	findAllErr error
}

func (m *mockSubRepoAll) Save(domain.Subscription) error { return nil }
func (m *mockSubRepoAll) FindByPhone(string) ([]domain.Subscription, error) {
	return nil, errors.New("not found")
}
func (m *mockSubRepoAll) FindAll() ([]domain.Subscription, error) {
	return m.all, m.findAllErr
}
func (m *mockSubRepoAll) Delete(string) error { return nil }
func (m *mockSubRepoAll) DeleteByEndpoint(ep string) error {
	m.deleted = append(m.deleted, ep)
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mondayAt8(year int, month time.Month, day int, loc *time.Location) time.Time {
	t := time.Date(year, month, day, 8, 0, 0, 0, loc)
	if t.Weekday() != time.Monday {
		panic("mondayAt8: provided date is not a Monday")
	}
	return t
}

func makeUC(rides []domain.Ride, subs []domain.Subscription, settings *mockSettingsRepo, notifier *mockNotifier, now time.Time) *usecase.SendEngagementReminder {
	rideRepo := &mockRideRepo{saved: rides}
	subRepo := &mockSubRepoAll{all: subs}
	loc := now.Location()
	uc := usecase.NewSendEngagementReminder(rideRepo, subRepo, settings, notifier, loc)
	uc.Clock = func() time.Time { return now }
	return uc
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestSendEngagementReminder_SendsWhenRidesLow(t *testing.T) {
	settings := newMockSettings()
	n := &mockNotifier{}
	subs := []domain.Subscription{
		{Phone: "555-0001", Endpoint: "https://push.example.com/1"},
	}
	// 5 rides — below threshold of 10
	rides := make([]domain.Ride, 5)

	uc := makeUC(rides, subs, settings, n, mondayAt8(2030, 1, 7, time.UTC))
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected push to be sent when rides < 10")
	}
	if len(settings.insertCalled) == 0 {
		t.Error("expected idempotency key to be stored after sending")
	}
}

func TestSendEngagementReminder_SkipsWhenRidesEnough(t *testing.T) {
	settings := newMockSettings()
	n := &mockNotifier{}
	subs := []domain.Subscription{
		{Phone: "555-0001", Endpoint: "https://push.example.com/1"},
	}
	// 10 rides — at threshold, no push needed
	rides := make([]domain.Ride, 10)

	uc := makeUC(rides, subs, settings, n, mondayAt8(2030, 1, 7, time.UTC))
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("expected no push when rides >= 10")
	}
}

func TestSendEngagementReminder_IdempotentWithinWeek(t *testing.T) {
	settings := newMockSettings()
	// Pre-populate the idempotency key for 2030-W02
	settings.data["engagement_reminder:2030-W02"] = "sent"

	n := &mockNotifier{}
	subs := []domain.Subscription{
		{Phone: "555-0001", Endpoint: "https://push.example.com/1"},
	}
	rides := make([]domain.Ride, 2) // would trigger send if key absent

	uc := makeUC(rides, subs, settings, n, mondayAt8(2030, 1, 7, time.UTC))
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("expected no push when already sent this week")
	}
}

func TestSendEngagementReminder_SkipsOutsideWindow(t *testing.T) {
	settings := newMockSettings()
	n := &mockNotifier{}
	subs := []domain.Subscription{{Phone: "555-0001", Endpoint: "https://push.example.com/1"}}
	rides := make([]domain.Ride, 0) // empty — would fire if in window

	// Tuesday — not a Monday
	tuesday := time.Date(2030, 1, 8, 8, 0, 0, 0, time.UTC)
	uc := makeUC(rides, subs, settings, n, tuesday)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("expected no push on a non-Monday")
	}

	// Monday before 08:00
	mondayBefore8 := time.Date(2030, 1, 7, 7, 59, 0, 0, time.UTC)
	uc2 := makeUC(rides, subs, settings, n, mondayBefore8)
	if err := uc2.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("expected no push on Monday before 08:00")
	}
}

func TestSendEngagementReminder_NoSubscribers_StillMarks(t *testing.T) {
	settings := newMockSettings()
	n := &mockNotifier{}
	rides := make([]domain.Ride, 3) // low enough to trigger

	uc := makeUC(rides, []domain.Subscription{}, settings, n, mondayAt8(2030, 2, 4, time.UTC))
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("expected no push with no subscribers")
	}
	if len(settings.insertCalled) == 0 {
		t.Error("expected idempotency key to be stored even with no subscribers")
	}
}

func TestSendEngagementReminder_Removes410GoneSubscription(t *testing.T) {
	settings := newMockSettings()
	gone := &goneNotifier{}
	subs := []domain.Subscription{
		{Phone: "555-0001", Endpoint: "https://push.example.com/stale"},
	}
	subRepo := &mockSubRepoAll{all: subs}
	rides := make([]domain.Ride, 1)
	rideRepo := &mockRideRepo{saved: rides}

	uc := usecase.NewSendEngagementReminder(rideRepo, subRepo, settings, gone, time.UTC)
	uc.Clock = func() time.Time { return mondayAt8(2030, 3, 4, time.UTC) }
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subRepo.deleted) == 0 {
		t.Error("expected stale 410 subscription to be pruned")
	}
}

func TestSendEngagementReminder_SettingsGetError(t *testing.T) {
	settings := newMockSettings()
	settings.getErr = errors.New("db error")
	n := &mockNotifier{}

	uc := makeUC(nil, nil, settings, n, mondayAt8(2030, 1, 7, time.UTC))
	err := uc.Execute()
	if err == nil {
		t.Error("expected error when settings.Get fails")
	}
	if n.called {
		t.Error("expected no push when settings check fails")
	}
}
