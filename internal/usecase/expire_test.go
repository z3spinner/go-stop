package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestExpireRides_CallsDeleteExpired(t *testing.T) {
	rides := &mockRideRepo{}
	uc := usecase.NewExpireRides(rides)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExpireRequests_CallsDeleteExpired(t *testing.T) {
	reqs := &mockRequestRepo{}
	uc := usecase.NewExpireRequests(reqs)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
