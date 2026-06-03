package usecase_test

import (
	"errors"
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestCancelInterest_DeletesPendingInterestForOwner(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "pending",
	}
	interests := &mockInterestRepo{byID: map[string]domain.Interest{"int-1": interest}}

	uc := usecase.NewCancelInterest(interests)
	if err := uc.Execute("int-1", "555-searcher"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(interests.deleteCalled) != 1 || interests.deleteCalled[0] != "int-1" {
		t.Errorf("expected Delete(int-1) to be called, got %v", interests.deleteCalled)
	}
}

func TestCancelInterest_RejectsNonOwner(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "pending",
	}
	interests := &mockInterestRepo{byID: map[string]domain.Interest{"int-1": interest}}

	uc := usecase.NewCancelInterest(interests)
	err := uc.Execute("int-1", "555-stranger")
	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if len(interests.deleteCalled) != 0 {
		t.Errorf("expected no delete, got %v", interests.deleteCalled)
	}
}

func TestCancelInterest_RejectsAlreadyAccepted(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{byID: map[string]domain.Interest{"int-1": interest}}

	uc := usecase.NewCancelInterest(interests)
	err := uc.Execute("int-1", "555-searcher")
	if !errors.Is(err, usecase.ErrNotPending) {
		t.Errorf("expected ErrNotPending, got %v", err)
	}
	if len(interests.deleteCalled) != 0 {
		t.Errorf("expected no delete, got %v", interests.deleteCalled)
	}
}

func TestCancelInterest_ReturnsErrorIfNotFound(t *testing.T) {
	interests := &mockInterestRepo{byID: map[string]domain.Interest{}}

	uc := usecase.NewCancelInterest(interests)
	if err := uc.Execute("missing", "555-searcher"); err == nil {
		t.Error("expected error for missing interest")
	}
}
