package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// mockRideRepoPendingFeedback controls FindPendingFeedback return value
type mockRideRepoPendingFeedback struct {
	pending []domain.Ride
}

func (m *mockRideRepoPendingFeedback) Save(domain.Ride) error              { return nil }
func (m *mockRideRepoPendingFeedback) FindByID(string) (domain.Ride, error) {
	return domain.Ride{}, nil
}
func (m *mockRideRepoPendingFeedback) FindAll() ([]domain.Ride, error)           { return nil, nil }
func (m *mockRideRepoPendingFeedback) FindByPhone(string) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoPendingFeedback) FindByOriginAndDestination(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoPendingFeedback) FindByOriginDestinationAndDate(string, string, time.Time) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoPendingFeedback) FindByOriginDestinationDateTime(string, string, time.Time, int) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoPendingFeedback) FindByOriginAndTime(string, string, time.Time, int) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoPendingFeedback) FindMatching(domain.Request) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoPendingFeedback) FindPendingFeedback() ([]domain.Ride, error) {
	return m.pending, nil
}
func (m *mockRideRepoPendingFeedback) Delete(string) error           { return nil }
func (m *mockRideRepoPendingFeedback) DeleteExpired() error          { return nil }
func (m *mockRideRepoPendingFeedback) SetFeedbackGiven(string) error { return nil }

func TestSendFeedbackReminders_SendsPushToDriver(t *testing.T) {
	rides := &mockRideRepoPendingFeedback{
		pending: []domain.Ride{
			{
				ID: "ride-1", DriverName: "Alice", Phone: "555-0001",
				Origin: "Saillans", Destination: "Crest",
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(rides, subs, n)
	err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected push notification to be sent")
	}
	if n.lastMsg.URL != "/rides/ride-1/feedback" {
		t.Errorf("expected URL /rides/ride-1/feedback, got %s", n.lastMsg.URL)
	}
}

func TestSendFeedbackReminders_SkipsIfNoSubscription(t *testing.T) {
	rides := &mockRideRepoPendingFeedback{
		pending: []domain.Ride{
			{ID: "ride-1", Phone: "555-no-sub"},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(rides, subs, n)
	err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send notification when driver has no subscription")
	}
}

func TestSendFeedbackReminders_NoPendingRides_NoNotifications(t *testing.T) {
	rides := &mockRideRepoPendingFeedback{pending: []domain.Ride{}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(rides, subs, n)
	err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send any notification with no pending rides")
	}
}
