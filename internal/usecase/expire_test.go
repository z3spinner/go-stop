package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type expiringRideRepo struct {
	called bool
}

func (r *expiringRideRepo) Save(domain.Ride) error                    { return nil }
func (r *expiringRideRepo) FindByID(string) (domain.Ride, error)      { return domain.Ride{}, nil }
func (r *expiringRideRepo) FindAll() ([]domain.Ride, error)           { return nil, nil }
func (r *expiringRideRepo) FindByPhone(string) ([]domain.Ride, error) { return nil, nil }
func (r *expiringRideRepo) FindByOriginAndDestination(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (r *expiringRideRepo) FindByOriginDestinationAndDate(string, string, time.Time) ([]domain.Ride, error) {
	return nil, nil
}
func (r *expiringRideRepo) FindByOriginDestinationDateTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (r *expiringRideRepo) FindByOriginAndTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (r *expiringRideRepo) FindByOriginAndDestinationFuzzy(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (r *expiringRideRepo) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (r *expiringRideRepo) Delete(string) error                                { return nil }
func (r *expiringRideRepo) DeleteExpired() error {
	r.called = true
	return nil
}
func (r *expiringRideRepo) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
func (r *expiringRideRepo) SetFeedbackGiven(string) error               { return nil }

type expiringRequestRepo struct {
	called bool
}

func (r *expiringRequestRepo) Save(domain.Request) error                          { return nil }
func (r *expiringRequestRepo) FindByPhone(string) ([]domain.Request, error)       { return nil, nil }
func (r *expiringRequestRepo) FindAllActive() ([]domain.Request, error)           { return nil, nil }
func (r *expiringRequestRepo) FindByID(string) (domain.Request, error)            { return domain.Request{}, nil }
func (r *expiringRequestRepo) FindMatching(domain.Ride) ([]domain.Request, error) { return nil, nil }
func (r *expiringRequestRepo) Delete(string) error                                { return nil }
func (r *expiringRequestRepo) DeleteExpired() error {
	r.called = true
	return nil
}

func TestExpireRides_CallsDeleteExpired(t *testing.T) {
	repo := &expiringRideRepo{}
	uc := usecase.NewExpireRides(repo)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called {
		t.Error("expected DeleteExpired to be called")
	}
}

func TestExpireRequests_CallsDeleteExpired(t *testing.T) {
	repo := &expiringRequestRepo{}
	uc := usecase.NewExpireRequests(repo)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called {
		t.Error("expected DeleteExpired to be called")
	}
}
