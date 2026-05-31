package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestGetRides_ReturnsAllRides(t *testing.T) {
	rides := &mockRideRepo{
		saved: []domain.Ride{
			{ID: "1", DriverName: "Alice", DepartureAt: time.Now()},
			{ID: "2", DriverName: "Bob", DepartureAt: time.Now()},
		},
	}
	uc := usecase.NewGetRides(rides)
	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 rides, got %d", len(result))
	}
}

func TestGetRides_ReturnsEmptySliceWhenNoRides(t *testing.T) {
	uc := usecase.NewGetRides(&mockRideRepo{})
	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected empty slice, not nil")
	}
}
