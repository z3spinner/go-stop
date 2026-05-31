package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockDestRepo struct {
	locations []string
}

func (m *mockDestRepo) GetAll() ([]string, error) { return m.locations, nil }

func TestGetDestinations_ReturnsAll(t *testing.T) {
	uc := usecase.NewGetDestinations(&mockDestRepo{locations: []string{"Village A", "Station", "Town B"}})
	result, err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 destinations, got %d", len(result))
	}
}
