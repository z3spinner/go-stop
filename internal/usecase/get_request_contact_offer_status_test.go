// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestGetRequestContactOfferStatus_ReturnsTrueWhenOfferExists(t *testing.T) {
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-searcher"},
		},
	}
	offers := &mockContactOfferRepo{
		saved: []domain.ContactOffer{{RequestID: "req-1", OffererPhone: "555-driver"}},
	}

	uc := usecase.NewGetRequestContactOfferStatus(reqRepo, offers)
	offered, err := uc.Execute("req-1", "555-driver")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !offered {
		t.Fatal("expected offered=true")
	}
}

func TestGetRequestContactOfferStatus_ReturnsFalseWhenOfferMissing(t *testing.T) {
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-searcher"},
		},
	}

	uc := usecase.NewGetRequestContactOfferStatus(reqRepo, &mockContactOfferRepo{})
	offered, err := uc.Execute("req-1", "555-driver")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if offered {
		t.Fatal("expected offered=false")
	}
}

func TestGetRequestContactOfferStatus_ReturnsNotFoundForUnknownRequest(t *testing.T) {
	uc := usecase.NewGetRequestContactOfferStatus(&mockRequestRepoByID{byID: map[string]domain.Request{}}, &mockContactOfferRepo{})
	_, err := uc.Execute("req-1", "555-driver")

	if !errors.Is(err, usecase.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetRequestContactOfferStatus_RejectsSearcherPhone(t *testing.T) {
	reqRepo := &mockRequestRepoByID{
		byID: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-searcher"},
		},
	}

	uc := usecase.NewGetRequestContactOfferStatus(reqRepo, &mockContactOfferRepo{})
	_, err := uc.Execute("req-1", "555-searcher")

	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}
