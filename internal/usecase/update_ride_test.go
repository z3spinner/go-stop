// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// editNotifQueue records which ride↔request pairs were marked sent, and reports
// pre-seeded pairs as already-enqueued (Enqueue → false) so we can assert that
// only newly-matching searchers are notified after an edit.
type editNotifQueue struct {
	existing map[string]bool
	marked   []string
}

func nqKey(rideID, reqID string) string { return rideID + "|" + reqID }

func (n *editNotifQueue) Enqueue(rideID, reqID, _ string) (bool, error) {
	if n.existing == nil {
		n.existing = map[string]bool{}
	}
	if n.existing[nqKey(rideID, reqID)] {
		return false, nil
	}
	n.existing[nqKey(rideID, reqID)] = true
	return true, nil
}
func (n *editNotifQueue) MarkSentByRideAndRequest(rideID, reqID string) error {
	n.marked = append(n.marked, nqKey(rideID, reqID))
	return nil
}
func (n *editNotifQueue) FindPending(time.Time, int) ([]domain.NotificationQueueEntry, error) {
	return nil, nil
}
func (n *editNotifQueue) MarkSent(string) error      { return nil }
func (n *editNotifQueue) DeleteForRide(string) error { return nil }
func (n *editNotifQueue) DeleteExpired() error       { return nil }
func (n *editNotifQueue) ListForSearcher(string) ([]domain.NotificationQueueEntry, error) {
	return nil, nil
}

func seededRide(id string) domain.Ride {
	return domain.Ride{
		ID: id, DriverName: "Alice", Phone: "555-0001",
		Origin: "Old Origin", Destination: "Old Dest",
		Date:          time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt:   time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility:   domain.Exact,
		PostedAt:      time.Date(2030, 5, 1, 8, 0, 0, 0, time.UTC),
		ExpiresAt:     time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
		FeedbackGiven: true,
	}
}

func TestUpdateRide_UpdatesFieldsAndPreservesIdentity(t *testing.T) {
	rides := &mockRideRepo{byID: map[string]domain.Ride{"ride-1": seededRide("ride-1")}}
	uc := usecase.NewUpdateRide(rides, &mockRequestRepo{}, &mockSubRepo{}, &noopNotifQueue{}, &mockNotifier{}, 60)

	newDeparture := time.Date(2030, 7, 2, 17, 30, 0, 0, time.UTC)
	got, err := uc.Execute("ride-1", "555-0001", "New Origin", "New Dest", newDeparture, domain.Approximate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Origin != "New Origin" || got.Destination != "New Dest" {
		t.Errorf("route not updated: %q → %q", got.Origin, got.Destination)
	}
	if !got.DepartureAt.Equal(newDeparture) {
		t.Errorf("departure not updated: %v", got.DepartureAt)
	}
	if got.Flexibility != domain.Approximate {
		t.Errorf("flexibility not updated: %d", got.Flexibility)
	}
	wantDate := time.Date(2030, 7, 2, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(wantDate) {
		t.Errorf("date not recomputed: got %v want %v", got.Date, wantDate)
	}
	wantExpiry := time.Date(2030, 7, 3, 0, 0, 0, 0, time.UTC)
	if !got.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("expiry not recomputed: got %v want %v", got.ExpiresAt, wantExpiry)
	}
	// Identity preserved.
	if got.ID != "ride-1" || got.DriverName != "Alice" || got.Phone != "555-0001" {
		t.Errorf("identity changed: %+v", got)
	}
	if !got.PostedAt.Equal(seededRide("ride-1").PostedAt) {
		t.Errorf("posted_at not preserved: %v", got.PostedAt)
	}
	if !got.FeedbackGiven {
		t.Error("feedback_given not preserved")
	}
}

func TestUpdateRide_NotFound(t *testing.T) {
	rides := &mockRideRepo{byID: map[string]domain.Ride{}}
	uc := usecase.NewUpdateRide(rides, &mockRequestRepo{}, &mockSubRepo{}, &noopNotifQueue{}, &mockNotifier{}, 60)
	_, err := uc.Execute("nope", "555-0001", "A", "B", time.Now(), domain.Exact)
	if !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateRide_WrongPhoneRejected(t *testing.T) {
	rides := &mockRideRepo{byID: map[string]domain.Ride{"ride-1": seededRide("ride-1")}}
	uc := usecase.NewUpdateRide(rides, &mockRequestRepo{}, &mockSubRepo{}, &noopNotifQueue{}, &mockNotifier{}, 60)
	_, err := uc.Execute("ride-1", "555-9999", "A", "B", time.Now(), domain.Exact)
	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestUpdateRide_DuplicateKeyConflict(t *testing.T) {
	rides := &mockRideRepo{
		byID:      map[string]domain.Ride{"ride-1": seededRide("ride-1")},
		updateErr: repository.ErrDuplicateRide,
	}
	uc := usecase.NewUpdateRide(rides, &mockRequestRepo{}, &mockSubRepo{}, &noopNotifQueue{}, &mockNotifier{}, 60)
	_, err := uc.Execute("ride-1", "555-0001", "A", "B", time.Now(), domain.Exact)
	if !errors.Is(err, repository.ErrDuplicateRide) {
		t.Errorf("expected ErrDuplicateRide, got %v", err)
	}
}

func TestUpdateRide_NotifiesOnlyNewlyMatchingSearchers(t *testing.T) {
	rides := &mockRideRepo{byID: map[string]domain.Ride{"ride-1": seededRide("ride-1")}}
	reqs := &mockRequestRepo{matching: []domain.Request{
		{ID: "reqA", Phone: "555-0002"}, // already notified for this ride
		{ID: "reqB", Phone: "555-0003"}, // newly matching after the edit
	}}
	queue := &editNotifQueue{existing: map[string]bool{nqKey("ride-1", "reqA"): true}}
	uc := usecase.NewUpdateRide(rides, reqs, &mockSubRepo{}, queue, &mockNotifier{}, 60)

	if _, err := uc.Execute("ride-1", "555-0001", "New Origin", "New Dest",
		time.Date(2030, 6, 1, 17, 0, 0, 0, time.UTC), domain.Approximate); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(queue.marked) != 1 || queue.marked[0] != nqKey("ride-1", "reqB") {
		t.Errorf("expected only reqB to be notified, got %v", queue.marked)
	}
}
