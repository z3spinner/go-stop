// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestContactOfferRepo_SaveAndListByRequest(t *testing.T) {
	truncate(t)
	repo := postgres.NewContactOfferRepo(testPool)

	requestID := uuid.New().String()
	offer := domain.ContactOffer{
		ID:           uuid.New().String(),
		RequestID:    requestID,
		OffererPhone: "0611000001",
		OffererName:  "Alice",
	}

	if err := repo.Save(offer); err != nil {
		t.Fatalf("Save: %v", err)
	}

	offers, err := repo.ListByRequest(requestID)
	if err != nil {
		t.Fatalf("ListByRequest: %v", err)
	}
	if len(offers) != 1 {
		t.Fatalf("expected 1 offer, got %d", len(offers))
	}
	got := offers[0]
	if got.OffererPhone != "0611000001" {
		t.Errorf("expected OffererPhone 0611000001, got %s", got.OffererPhone)
	}
	if got.OffererName != "Alice" {
		t.Errorf("expected OffererName Alice, got %s", got.OffererName)
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestContactOfferRepo_FindByRequestAndOfferer(t *testing.T) {
	truncate(t)
	repo := postgres.NewContactOfferRepo(testPool)

	requestID := uuid.New().String()
	offerID := uuid.New().String()

	if err := repo.Save(domain.ContactOffer{
		ID:           offerID,
		RequestID:    requestID,
		OffererPhone: "0611000001",
		OffererName:  "Alice",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	found, err := repo.FindByRequestAndOfferer(requestID, "0611000001")
	if err != nil {
		t.Fatalf("FindByRequestAndOfferer: %v", err)
	}
	if found.ID != offerID {
		t.Errorf("expected ID %s, got %s", offerID, found.ID)
	}
}

func TestContactOfferRepo_FindByRequestAndOfferer_NotFound(t *testing.T) {
	truncate(t)
	repo := postgres.NewContactOfferRepo(testPool)

	_, err := repo.FindByRequestAndOfferer(uuid.New().String(), "0611000001")
	if err == nil {
		t.Error("expected error for missing offer")
	}
}

func TestContactOfferRepo_Save_DeduplicatesOnConflict(t *testing.T) {
	truncate(t)
	repo := postgres.NewContactOfferRepo(testPool)

	requestID := uuid.New().String()
	base := domain.ContactOffer{
		ID:           uuid.New().String(),
		RequestID:    requestID,
		OffererPhone: "0611000001",
		OffererName:  "Alice",
	}

	if err := repo.Save(base); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	// Same (request_id, offerer_phone) — ON CONFLICT DO NOTHING.
	duplicate := base
	duplicate.ID = uuid.New().String()
	if err := repo.Save(duplicate); err != nil {
		t.Fatalf("duplicate Save: %v", err)
	}

	offers, err := repo.ListByRequest(requestID)
	if err != nil {
		t.Fatalf("ListByRequest: %v", err)
	}
	if len(offers) != 1 {
		t.Errorf("expected exactly 1 offer after duplicate Save, got %d", len(offers))
	}
}

func TestContactOfferRepo_ListByRequest_MultipleOfferers(t *testing.T) {
	truncate(t)
	repo := postgres.NewContactOfferRepo(testPool)

	requestID := uuid.New().String()
	now := time.Now()

	for _, o := range []domain.ContactOffer{
		{ID: uuid.New().String(), RequestID: requestID, OffererPhone: "0611000001", OffererName: "Alice"},
		{ID: uuid.New().String(), RequestID: requestID, OffererPhone: "0622000002", OffererName: "Bob"},
		{ID: uuid.New().String(), RequestID: requestID, OffererPhone: "0633000003", OffererName: "Carol"},
	} {
		_ = now // suppress unused-variable warning
		if err := repo.Save(o); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	offers, err := repo.ListByRequest(requestID)
	if err != nil {
		t.Fatalf("ListByRequest: %v", err)
	}
	if len(offers) != 3 {
		t.Errorf("expected 3 offers, got %d", len(offers))
	}
}

func TestContactOfferRepo_ListByRequest_EmptyForOtherRequest(t *testing.T) {
	truncate(t)
	repo := postgres.NewContactOfferRepo(testPool)

	requestID := uuid.New().String()
	otherID := uuid.New().String()

	if err := repo.Save(domain.ContactOffer{
		ID:           uuid.New().String(),
		RequestID:    requestID,
		OffererPhone: "0611000001",
		OffererName:  "Alice",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	offers, err := repo.ListByRequest(otherID)
	if err != nil {
		t.Fatalf("ListByRequest: %v", err)
	}
	if len(offers) != 0 {
		t.Errorf("expected 0 offers for different request, got %d", len(offers))
	}
}
