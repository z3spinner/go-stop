// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// mockContactOfferRepo implements repository.ContactOfferRepository in memory.
type mockContactOfferRepo struct {
	saved   []domain.ContactOffer
	saveErr error
}

func (m *mockContactOfferRepo) Save(o domain.ContactOffer) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, o)
	return nil
}

func (m *mockContactOfferRepo) FindByRequestAndOfferer(requestID, offererPhone string) (domain.ContactOffer, error) {
	for _, o := range m.saved {
		if o.RequestID == requestID && o.OffererPhone == offererPhone {
			return o, nil
		}
	}
	return domain.ContactOffer{}, errors.New("not found")
}

func (m *mockContactOfferRepo) ListByRequest(requestID string) ([]domain.ContactOffer, error) {
	var out []domain.ContactOffer
	for _, o := range m.saved {
		if o.RequestID == requestID {
			out = append(out, o)
		}
	}
	return out, nil
}

// ── OfferContact tests ────────────────────────────────────────────────────────

func TestOfferContact_CreatesOfferAndNotifiesSearcher(t *testing.T) {
	requests := &mockRequestRepo{}
	requests.saved = append(requests.saved, domain.Request{
		ID:          "req-1",
		Phone:       "555-searcher",
		SearcherName: "Bob",
		Origin:      "Saillans",
		Destination: "Crest",
	})
	// Override FindByID to return the saved request.
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {
				ID:          "req-1",
				Phone:       "555-searcher",
				SearcherName: "Bob",
				Origin:      "Saillans",
				Destination: "Crest",
			},
		},
	}
	offers := &mockContactOfferRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-searcher": {Phone: "555-searcher", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewOfferContact(reqRepo, offers, subs, n)
	created, err := uc.Execute("req-1", "555-offerer", "Alice")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected created=true for a new offer")
	}
	if len(offers.saved) != 1 {
		t.Fatalf("expected 1 saved offer, got %d", len(offers.saved))
	}
	if offers.saved[0].OffererPhone != "555-offerer" {
		t.Errorf("expected offerer phone 555-offerer, got %s", offers.saved[0].OffererPhone)
	}
	if !n.called {
		t.Error("expected push notification sent to searcher")
	}
}

func TestOfferContact_RejectsEmptyName(t *testing.T) {
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-searcher"},
		},
	}
	offers := &mockContactOfferRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewOfferContact(reqRepo, offers, subs, n)
	_, err := uc.Execute("req-1", "555-offerer", "")

	if !errors.Is(err, usecase.ErrNameRequired) {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}
	if len(offers.saved) != 0 {
		t.Error("no offer should be saved when name is empty")
	}
}

func TestOfferContact_RejectsSearcherOfferingToThemself(t *testing.T) {
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-same"},
		},
	}
	offers := &mockContactOfferRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewOfferContact(reqRepo, offers, subs, n)
	_, err := uc.Execute("req-1", "555-same", "Self")

	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestOfferContact_ReturnsErrorIfRequestNotFound(t *testing.T) {
	reqRepo := &mockRequestRepoByID{byID: map[string]domain.Request{}}
	offers := &mockContactOfferRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewOfferContact(reqRepo, offers, subs, n)
	_, err := uc.Execute("nonexistent", "555-offerer", "Alice")

	if err == nil {
		t.Error("expected error for missing request")
	}
}

func TestOfferContact_DuplicateOfferRenotifiesButReturnsCreatedFalse(t *testing.T) {
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-searcher", Origin: "A", Destination: "B"},
		},
	}
	offers := &mockContactOfferRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewOfferContact(reqRepo, offers, subs, n)

	created1, err := uc.Execute("req-1", "555-offerer", "Alice")
	if err != nil || !created1 {
		t.Fatalf("first offer should succeed with created=true, got err=%v created=%v", err, created1)
	}

	created2, err := uc.Execute("req-1", "555-offerer", "Alice")
	if err != nil {
		t.Fatalf("second (duplicate) offer should not error: %v", err)
	}
	if created2 {
		t.Error("duplicate offer should return created=false")
	}
	if len(offers.saved) != 1 {
		t.Errorf("expected only 1 saved offer after duplicate, got %d", len(offers.saved))
	}
}

// ── GetRequestContactOffers tests ─────────────────────────────────────────────

func TestGetRequestContactOffers_ReturnsOffersForOwner(t *testing.T) {
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-owner"},
		},
	}
	offers := &mockContactOfferRepo{
		saved: []domain.ContactOffer{
			{ID: "off-1", RequestID: "req-1", OffererPhone: "555-a", OffererName: "Alice"},
			{ID: "off-2", RequestID: "req-1", OffererPhone: "555-b", OffererName: "Bob"},
		},
	}

	uc := usecase.NewGetRequestContactOffers(reqRepo, offers)
	result, err := uc.Execute("req-1", "555-owner")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 offers, got %d", len(result))
	}
}

func TestGetRequestContactOffers_RejectsNonOwner(t *testing.T) {
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-owner"},
		},
	}
	offers := &mockContactOfferRepo{}

	uc := usecase.NewGetRequestContactOffers(reqRepo, offers)
	_, err := uc.Execute("req-1", "555-other")

	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestGetRequestContactOffers_ReturnsErrorIfRequestNotFound(t *testing.T) {
	reqRepo := &mockRequestRepoByID{byID: map[string]domain.Request{}}
	offers := &mockContactOfferRepo{}

	uc := usecase.NewGetRequestContactOffers(reqRepo, offers)
	_, err := uc.Execute("nonexistent", "555-owner")

	if err == nil {
		t.Error("expected error for missing request")
	}
}

// ── mockRequestRepoByID — minimal request repo for contact-offer tests ─────────
// The existing mockRequestRepo doesn't support FindByID with stored entries, so
// we define a focused mock here that only implements what OfferContact/
// GetRequestContactOffers require.

type mockRequestRepoByID struct {
	byID map[string]domain.Request
}

func (m *mockRequestRepoByID) Save(r domain.Request) error { return nil }
func (m *mockRequestRepoByID) FindByID(id string) (domain.Request, error) {
	r, ok := m.byID[id]
	if !ok {
		return domain.Request{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRequestRepoByID) FindByPhone(string) ([]domain.Request, error) { return nil, nil }
func (m *mockRequestRepoByID) FindAllActive() ([]domain.Request, error)     { return nil, nil }
func (m *mockRequestRepoByID) FindMatching(domain.Ride) ([]domain.Request, error) {
	return nil, nil
}
func (m *mockRequestRepoByID) Delete(string) error  { return nil }
func (m *mockRequestRepoByID) DeleteExpired() error { return nil }
