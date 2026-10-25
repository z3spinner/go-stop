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

// ── mock AlertExpiryRepository ────────────────────────────────────────────────

type mockAlertExpiryRepo struct {
	due        []domain.Request
	findErr    error
	findCalled bool
	before     time.Time
	warned     map[string]time.Time // id → warned expiry
	setID      string
	setExpiry  time.Time
	setErr     error
}

func (m *mockAlertExpiryRepo) FindExpiringBefore(before time.Time) ([]domain.Request, error) {
	m.findCalled = true
	m.before = before
	return m.due, m.findErr
}

func (m *mockAlertExpiryRepo) MarkExpiryWarned(id string, expiresAt time.Time) error {
	if m.warned == nil {
		m.warned = map[string]time.Time{}
	}
	m.warned[id] = expiresAt
	return nil
}

func (m *mockAlertExpiryRepo) SetExpiry(id string, expiresAt time.Time) error {
	m.setID, m.setExpiry = id, expiresAt
	return m.setErr
}

func within(t *testing.T, what string, got, want time.Time) {
	t.Helper()
	if d := got.Sub(want); d < -time.Minute || d > time.Minute {
		t.Errorf("%s = %v, want ≈ %v", what, got, want)
	}
}

// ── SendAlertExpiryWarnings ───────────────────────────────────────────────────

func TestSendAlertExpiryWarnings_WarnsOncePerExpiry(t *testing.T) {
	exp := time.Date(2030, 6, 10, 9, 0, 0, 0, time.UTC)
	alerts := &mockAlertExpiryRepo{due: []domain.Request{
		{ID: "req-1", Phone: "555-0001", Origin: "Crest", Destination: "Saillans", ExpiresAt: exp},
	}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	if err := usecase.NewSendAlertExpiryWarnings(alerts, subs, n, usecase.QuietHours{}, time.UTC).Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Looks AlertExpiryNotice ahead.
	within(t, "lookahead", alerts.before, time.Now().Add(domain.AlertExpiryNotice))
	if !n.called || n.lastMsg.URL != "/my-searches?alert=req-1" || n.lastMsg.InboxID == "" {
		t.Errorf("unexpected push: %+v", n.lastMsg)
	}
	assertInbox(t, n, "555-0001", domain.Message{
		Kind:      domain.KindAlertExpiring,
		DedupKey:  "alert_expiring:req-1@2030-06-10",
		RequestID: "req-1",
	})
	if got, ok := alerts.warned["req-1"]; !ok || !got.Equal(exp) {
		t.Errorf("expected req-1 marked warned for %v, got %v (ok=%v)", exp, got, ok)
	}
}

func TestSendAlertExpiryWarnings_NoDeviceStillReachesTheInbox(t *testing.T) {
	alerts := &mockAlertExpiryRepo{due: []domain.Request{{ID: "req-1", Phone: "555-no-sub", ExpiresAt: time.Now().Add(time.Hour)}}}
	n := &mockNotifier{}
	if err := usecase.NewSendAlertExpiryWarnings(alerts, &mockSubRepo{subs: map[string]domain.Subscription{}}, n, usecase.QuietHours{}, time.UTC).Execute(); err != nil {
		t.Fatal(err)
	}
	if n.called {
		t.Error("no device: nothing to push")
	}
	if len(n.recorded) != 1 || len(alerts.warned) != 1 {
		t.Errorf("expected the warning recorded and marked, got records=%d warned=%v", len(n.recorded), alerts.warned)
	}
}

func TestSendAlertExpiryWarnings_WaitsDuringQuietHours(t *testing.T) {
	h := time.Now().UTC().Hour()
	quiet := usecase.QuietHours{Start: h, End: (h + 1) % 24}
	alerts := &mockAlertExpiryRepo{due: []domain.Request{{ID: "req-1", Phone: "555-0001"}}}
	n := &mockNotifier{}
	if err := usecase.NewSendAlertExpiryWarnings(alerts, &mockSubRepo{}, n, quiet, time.UTC).Execute(); err != nil {
		t.Fatal(err)
	}
	if alerts.findCalled || len(n.recorded) != 0 || len(alerts.warned) != 0 {
		t.Errorf("quiet hours: nothing should be sent, got find=%v records=%d warned=%v", alerts.findCalled, len(n.recorded), alerts.warned)
	}
}

func TestSendAlertExpiryWarnings_FindError(t *testing.T) {
	alerts := &mockAlertExpiryRepo{findErr: errors.New("db down")}
	if err := usecase.NewSendAlertExpiryWarnings(alerts, &mockSubRepo{}, &mockNotifier{}, usecase.QuietHours{}, time.UTC).Execute(); err == nil {
		t.Error("expected the repository error")
	}
}

// ── RenewRequest ──────────────────────────────────────────────────────────────

func TestRenewRequest_ExtendsAnOpenEndedAlertAndAcknowledgesTheWarning(t *testing.T) {
	oldExp := time.Date(2030, 6, 10, 9, 0, 0, 0, time.UTC)
	cases := map[string]domain.Request{
		"anytime": {ID: "req-1", Phone: "555-0001", ExpiresAt: oldExp},
		"daily":   {ID: "req-1", Phone: "555-0001", DepartureAt: time.Date(1970, 1, 1, 7, 35, 0, 0, time.UTC), ExpiresAt: oldExp},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			reqs := &mockRequestRepoByID{byID: map[string]domain.Request{"req-1": req}}
			alerts := &mockAlertExpiryRepo{}
			n := &mockNotifier{}

			got, err := usecase.NewRenewRequest(reqs, alerts, n).Execute("req-1", "555-0001")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := time.Now().Add(domain.AlertLifetime)
			within(t, "returned ExpiresAt", got.ExpiresAt, want)
			if alerts.setID != "req-1" || !alerts.setExpiry.Equal(got.ExpiresAt) {
				t.Errorf("SetExpiry(%q, %v), want (req-1, %v)", alerts.setID, alerts.setExpiry, got.ExpiresAt)
			}
			if len(n.acked) != 1 || n.acked[0] != "555-0001/alert_expiring:req-1@2030-06-10" {
				t.Errorf("acked = %v, want the warning for the old expiry", n.acked)
			}
		})
	}
}

func TestRenewRequest_Rejects(t *testing.T) {
	dated := domain.Request{ID: "req-1", Phone: "555-0001",
		Date: time.Date(2030, 6, 5, 0, 0, 0, 0, time.UTC), DepartureAt: time.Date(2030, 6, 5, 8, 0, 0, 0, time.UTC)}
	anytime := domain.Request{ID: "req-1", Phone: "555-0001"}
	cases := []struct {
		name  string
		req   domain.Request
		id    string
		phone string
		want  error
	}{
		{"unknown request", anytime, "req-x", "555-0001", usecase.ErrNotFound},
		{"someone else's alert", anytime, "req-1", "555-9999", usecase.ErrUnauthorized},
		{"no phone", anytime, "req-1", "", usecase.ErrUnauthorized},
		{"dated request", dated, "req-1", "555-0001", usecase.ErrNotRenewable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqs := &mockRequestRepoByID{byID: map[string]domain.Request{"req-1": tc.req}}
			alerts := &mockAlertExpiryRepo{}
			n := &mockNotifier{}
			_, err := usecase.NewRenewRequest(reqs, alerts, n).Execute(tc.id, tc.phone)
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
			if alerts.setID != "" || len(n.acked) != 0 {
				t.Errorf("nothing should change, got set=%q acked=%v", alerts.setID, n.acked)
			}
		})
	}
}

func TestRenewRequest_StoreError(t *testing.T) {
	reqs := &mockRequestRepoByID{byID: map[string]domain.Request{"req-1": {ID: "req-1", Phone: "555-0001"}}}
	n := &mockNotifier{}
	_, err := usecase.NewRenewRequest(reqs, &mockAlertExpiryRepo{setErr: errors.New("db down")}, n).Execute("req-1", "555-0001")
	if err == nil {
		t.Error("expected the store error")
	}
	if len(n.acked) != 0 {
		t.Error("the warning stays unread when the renewal failed")
	}
}
