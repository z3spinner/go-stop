// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

// mockFeedbackQueue is the shared in-memory FeedbackQueueRepository for usecase tests.
type mockFeedbackQueue struct {
	enqueueCalled     bool
	enqueueWindowArg  time.Time
	enqueueErr        error
	due               []domain.FeedbackTask
	byRideID          map[string]domain.FeedbackTask
	claimedRides      map[string]bool // ride IDs already claimed via DeleteByRideID
	marked            []string
	deletedByRideID   []string
	deleteExhaustedOK bool
}

func (m *mockFeedbackQueue) EnqueueStartedRides(windowStartAfter time.Time) error {
	m.enqueueCalled = true
	m.enqueueWindowArg = windowStartAfter
	return m.enqueueErr
}
func (m *mockFeedbackQueue) FindDue(retryAfter time.Time, maxRetries int) ([]domain.FeedbackTask, error) {
	return m.due, nil
}
func (m *mockFeedbackQueue) FindByRideID(rideID string) (domain.FeedbackTask, error) {
	if m.byRideID != nil {
		if t, ok := m.byRideID[rideID]; ok {
			return t, nil
		}
	}
	return domain.FeedbackTask{}, errors.New("not found")
}
func (m *mockFeedbackQueue) MarkSent(id string) error {
	m.marked = append(m.marked, id)
	return nil
}

// DeleteByRideID models the atomic claim: the first call for a ride that has a
// queued task returns true (claimed); later calls return false. byRideID is left
// intact so FindByRideID still serves the phone check, modelling the real race
// where concurrent callers both read the task before either claims it.
func (m *mockFeedbackQueue) DeleteByRideID(rideID string) (bool, error) {
	m.deletedByRideID = append(m.deletedByRideID, rideID)
	if m.byRideID == nil {
		return false, nil
	}
	if _, ok := m.byRideID[rideID]; !ok {
		return false, nil
	}
	if m.claimedRides[rideID] {
		return false, nil
	}
	if m.claimedRides == nil {
		m.claimedRides = map[string]bool{}
	}
	m.claimedRides[rideID] = true
	return true, nil
}
func (m *mockFeedbackQueue) DeleteExhausted(maxRetries int, ttl time.Duration) error {
	m.deleteExhaustedOK = true
	return nil
}
