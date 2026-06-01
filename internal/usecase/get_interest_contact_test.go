package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestGetInterestContact_DriverGetsSearcherPhone(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver", DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetInterestContact(interests, rides)
	phone, err := uc.Execute("int-1", "555-driver")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phone != "555-searcher" {
		t.Errorf("expected 555-searcher, got %s", phone)
	}
}

func TestGetInterestContact_SearcherGetsDriverPhone(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver", DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetInterestContact(interests, rides)
	phone, err := uc.Execute("int-1", "555-searcher")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phone != "555-driver" {
		t.Errorf("expected 555-driver, got %s", phone)
	}
}

func TestGetInterestContact_ReturnsErrorIfPending(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "pending",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver", DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetInterestContact(interests, rides)
	_, err := uc.Execute("int-1", "555-driver")

	if err == nil {
		t.Error("expected error for pending interest")
	}
}

func TestGetInterestContact_RejectsUnauthorizedPhone(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver", DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetInterestContact(interests, rides)
	_, err := uc.Execute("int-1", "555-stranger")

	if err == nil {
		t.Error("expected error for unauthorized phone")
	}
}
