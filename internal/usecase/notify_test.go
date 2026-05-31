package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockNotifier struct {
	called  bool
	lastMsg domain.Message
	err     error
}

func (m *mockNotifier) Send(sub domain.Subscription, msg domain.Message) error {
	m.called = true
	m.lastMsg = msg
	return m.err
}

func TestNotifySearcher_SendsCorrectMessage(t *testing.T) {
	n := &mockNotifier{}
	sub := domain.Subscription{Phone: "555-0001"}
	ride := domain.Ride{
		ID: "ride-1", DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Train Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	}
	err := usecase.NotifySearcher(sub, ride, n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("notifier.Send was not called")
	}
	if n.lastMsg.URL != "/rides/ride-1" {
		t.Errorf("expected URL /rides/ride-1, got %s", n.lastMsg.URL)
	}
	if n.lastMsg.ContactName != "Alice" {
		t.Errorf("expected ContactName Alice, got %s", n.lastMsg.ContactName)
	}
}

func TestNotifyDriver_SendsCorrectMessage(t *testing.T) {
	n := &mockNotifier{}
	sub := domain.Subscription{Phone: "555-0002"}
	req := domain.Request{
		ID: "req-1", SearcherName: "Bob", Phone: "555-0002",
		Origin: "Village A", Destination: "Train Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	}
	err := usecase.NotifyDriver(sub, req, n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.lastMsg.URL != "/requests/req-1" {
		t.Errorf("expected URL /requests/req-1, got %s", n.lastMsg.URL)
	}
	if n.lastMsg.ContactName != "Bob" {
		t.Errorf("expected ContactName Bob, got %s", n.lastMsg.ContactName)
	}
}

func TestNotifySearcher_PropagatesNotifierError(t *testing.T) {
	n := &mockNotifier{err: errors.New("push failed")}
	err := usecase.NotifySearcher(domain.Subscription{}, domain.Ride{ID: "x"}, n)
	if err == nil {
		t.Error("expected error to be propagated")
	}
}

func TestNotifyDriver_PropagatesNotifierError(t *testing.T) {
	n := &mockNotifier{err: errors.New("push failed")}
	err := usecase.NotifyDriver(domain.Subscription{}, domain.Request{ID: "x"}, n)
	if err == nil {
		t.Error("expected error to be propagated")
	}
}
