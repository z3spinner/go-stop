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

type mockRideRepoDelete struct {
	rides   map[string]domain.Ride
	deleted []string
}

func (m *mockRideRepoDelete) Save(domain.Ride) error { return nil }
func (m *mockRideRepoDelete) FindByID(id string) (domain.Ride, error) {
	r, ok := m.rides[id]
	if !ok {
		return domain.Ride{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRideRepoDelete) FindAll() ([]domain.Ride, error)           { return nil, nil }
func (m *mockRideRepoDelete) FindByPhone(string) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoDelete) FindByOriginAndDestination(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoDelete) FindByOriginDestinationAndDate(string, string, time.Time) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoDelete) FindByOriginDestinationDateTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoDelete) FindByOriginAndTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoDelete) FindByOriginAndDestinationFuzzy(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoDelete) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoDelete) Delete(id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *mockRideRepoDelete) DeleteExpired() error                        { return nil }
func (m *mockRideRepoDelete) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoDelete) SetFeedbackGiven(string) error               { return nil }

type mockRequestRepoDelete struct {
	requests map[string]domain.Request
	deleted  []string
}

func (m *mockRequestRepoDelete) Save(domain.Request) error                    { return nil }
func (m *mockRequestRepoDelete) FindByPhone(string) ([]domain.Request, error) { return nil, nil }
func (m *mockRequestRepoDelete) FindAllActive() ([]domain.Request, error)     { return nil, nil }
func (m *mockRequestRepoDelete) FindByID(id string) (domain.Request, error) {
	r, ok := m.requests[id]
	if !ok {
		return domain.Request{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRequestRepoDelete) FindMatching(domain.Ride) ([]domain.Request, error) { return nil, nil }
func (m *mockRequestRepoDelete) Delete(id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *mockRequestRepoDelete) DeleteExpired() error { return nil }

type noopNotifQueue struct{}

func (n *noopNotifQueue) Enqueue(string, string, string) error { return nil }
func (n *noopNotifQueue) FindPending(time.Time, int) ([]domain.NotificationQueueEntry, error) {
	return nil, nil
}
func (n *noopNotifQueue) MarkSent(string) error                         { return nil }
func (n *noopNotifQueue) MarkSentByRideAndRequest(string, string) error { return nil }
func (n *noopNotifQueue) DeleteForRide(string) error                    { return nil }
func (n *noopNotifQueue) DeleteExpired() error                          { return nil }
func (n *noopNotifQueue) ListForSearcher(string) ([]domain.NotificationQueueEntry, error) {
	return nil, nil
}
func TestDeleteRide_DeletesWhenPhoneMatches(t *testing.T) {
	rides := &mockRideRepoDelete{
		rides: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-0001"}},
	}
	uc := usecase.NewDeleteRide(rides, &noopNotifQueue{})
	if err := uc.Execute("ride-1", "555-0001"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rides.deleted) != 1 || rides.deleted[0] != "ride-1" {
		t.Error("expected ride-1 to be deleted")
	}
}

func TestDeleteRide_RejectsWrongPhone(t *testing.T) {
	rides := &mockRideRepoDelete{
		rides: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-0001"}},
	}
	uc := usecase.NewDeleteRide(rides, &noopNotifQueue{})
	if err := uc.Execute("ride-1", "555-9999"); err == nil {
		t.Error("expected unauthorized error")
	}
	if len(rides.deleted) != 0 {
		t.Error("ride should not have been deleted")
	}
}

func TestDeleteRide_ReturnsErrorIfNotFound(t *testing.T) {
	uc := usecase.NewDeleteRide(&mockRideRepoDelete{rides: map[string]domain.Ride{}}, &noopNotifQueue{})
	if err := uc.Execute("nonexistent", "555-0001"); err == nil {
		t.Error("expected not found error")
	}
}

func TestDeleteRequest_DeletesWhenPhoneMatches(t *testing.T) {
	reqs := &mockRequestRepoDelete{
		requests: map[string]domain.Request{"req-1": {ID: "req-1", Phone: "555-0002"}},
	}
	uc := usecase.NewDeleteRequest(reqs)
	if err := uc.Execute("req-1", "555-0002"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs.deleted) != 1 {
		t.Error("expected req-1 to be deleted")
	}
}

func TestDeleteRequest_RejectsWrongPhone(t *testing.T) {
	reqs := &mockRequestRepoDelete{
		requests: map[string]domain.Request{"req-1": {ID: "req-1", Phone: "555-0002"}},
	}
	uc := usecase.NewDeleteRequest(reqs)
	if err := uc.Execute("req-1", "555-9999"); err == nil {
		t.Error("expected unauthorized error")
	}
}

func TestDeleteRequest_ReturnsErrorIfNotFound(t *testing.T) {
	reqs := &mockRequestRepoDelete{requests: map[string]domain.Request{}}
	uc := usecase.NewDeleteRequest(reqs)
	if err := uc.Execute("nonexistent", "555-0001"); err == nil {
		t.Error("expected not found error")
	}
}
