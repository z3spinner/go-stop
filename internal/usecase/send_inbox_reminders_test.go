// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestParseQuietHours(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want usecase.QuietHours
		err  bool
	}{
		{"21-8", usecase.QuietHours{Start: 21, End: 8}, false},
		{" 22 - 7 ", usecase.QuietHours{Start: 22, End: 7}, false},
		{"", usecase.QuietHours{}, false},
		{"none", usecase.QuietHours{}, false},
		{"OFF", usecase.QuietHours{}, false},
		{"25-8", usecase.QuietHours{}, true},
		{"night", usecase.QuietHours{}, true},
	} {
		got, err := usecase.ParseQuietHours(tc.in)
		if (err != nil) != tc.err || got != tc.want {
			t.Errorf("ParseQuietHours(%q) = %+v, %v; want %+v, err=%v", tc.in, got, err, tc.want, tc.err)
		}
	}
}

func TestQuietHours_Contains(t *testing.T) {
	night := usecase.QuietHours{Start: 21, End: 8}
	for h, want := range map[int]bool{20: false, 21: true, 23: true, 0: true, 7: true, 8: false, 12: false} {
		if got := night.Contains(h); got != want {
			t.Errorf("21-8 contains %d = %v, want %v", h, got, want)
		}
	}
	lunch := usecase.QuietHours{Start: 12, End: 14}
	if !lunch.Contains(13) || lunch.Contains(14) || lunch.Contains(11) {
		t.Error("non-wrapping window wrong")
	}
	if (usecase.QuietHours{}).Contains(3) {
		t.Error("an empty window is never quiet")
	}
}

func TestSendInboxReminders_PushesDueItemsWithoutRecording(t *testing.T) {
	inbox := &mockInboxRepo{due: []domain.InboxItem{
		{ID: "n1", Phone: "0611", Kind: domain.KindInterestReceived, Title: "Quelqu'un demande votre contact",
			Body: "Bob cherche un trajet A → B", URL: "/my-rides"},
		{ID: "n2", Phone: "0622", Kind: domain.KindContactOffered, Title: "Alice can help you!", URL: "/my-searches"},
	}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"0611": {Phone: "0611", Endpoint: "https://push.example.com/1"},
		"0622": {Phone: "0622", Endpoint: "https://push.example.com/2"},
	}}
	n := &mockNotifier{}
	before := time.Now()

	uc := usecase.NewSendInboxReminders(inbox, subs, n, 3, usecase.QuietHours{}, time.UTC)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// One query per rule, due 3h after their last push.
	if len(inbox.dueCalls) != len(domain.ReminderRules) {
		t.Errorf("calls = %v, want one per rule", inbox.dueCalls)
	}
	want := before.Add(-3 * time.Hour)
	if d := inbox.dueCutoff.Sub(want); d < 0 || d > time.Minute {
		t.Errorf("cutoff %v, want ≈ %v", inbox.dueCutoff, want)
	}
	if len(n.sent) != 2 {
		t.Fatalf("expected 2 pushes, got %d", len(n.sent))
	}
	first := n.sent[0]
	if first.Title != "Rappel : Quelqu'un demande votre contact" || first.URL != "/my-rides" || first.InboxID != "n1" {
		t.Errorf("unexpected reminder push: %+v", first)
	}
	// Push only: nothing new in anyone's inbox.
	if len(n.recorded) != 0 {
		t.Errorf("reminders must not create inbox items, got %d records", len(n.recorded))
	}
	if len(inbox.reminded) != 2 || inbox.reminded[0] != "n1" || inbox.reminded[1] != "n2" {
		t.Errorf("reminded = %v", inbox.reminded)
	}
}

func TestReminderRules(t *testing.T) {
	rules := map[string]int{}
	for _, r := range domain.ReminderRules {
		for _, k := range r.Kinds {
			rules[k] = r.MaxReminders
		}
	}
	// Notifications about another person's action: until handled.
	for _, k := range []string{domain.KindInterestReceived, domain.KindInterestAccepted, domain.KindDriverShared, domain.KindContactOffered} {
		if max, ok := rules[k]; !ok || max != 0 {
			t.Errorf("%s: want unlimited reminders, got ok=%v max=%d", k, ok, max)
		}
	}
	// Ride matches and the feedback prompt: 2 reminders, 3 pushes in all (as
	// their old retries did).
	for _, k := range []string{domain.KindRideMatch, domain.KindFeedbackReminder} {
		if max := rules[k]; max != 2 {
			t.Errorf("%s: want 2 reminders, got %d", k, max)
		}
	}
	// No reminders for a driver's match (informational), cancellations, tests.
	for _, k := range []string{domain.KindRequestMatch, domain.KindInterestCancelled, domain.KindTest} {
		if _, ok := rules[k]; ok {
			t.Errorf("%s must not get inbox reminders", k)
		}
	}
}

func TestSendInboxReminders_RemindsTheFeedbackPromptWithItsCap(t *testing.T) {
	inbox := &mockInboxRepo{due: []domain.InboxItem{
		{ID: "f1", Phone: "0611", Kind: domain.KindFeedbackReminder, Title: "Votre trajet est-il parti avec des passagers ?", URL: "/rides/r1/feedback"},
	}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{"0611": {Phone: "0611", Endpoint: "https://push.example.com/1"}}}
	n := &mockNotifier{}
	if err := usecase.NewSendInboxReminders(inbox, subs, n, 3, usecase.QuietHours{}, time.UTC).Execute(); err != nil {
		t.Fatal(err)
	}
	var capped *dueCall
	for i, c := range inbox.dueCalls {
		if len(c.kinds) == 1 && c.kinds[0] == domain.KindFeedbackReminder {
			capped = &inbox.dueCalls[i]
		}
	}
	if capped == nil || capped.max != domain.FeedbackPromptMaxReminders {
		t.Fatalf("feedback prompt must be queried with its cap, calls=%v", inbox.dueCalls)
	}
	if len(n.sent) != 1 || n.sent[0].Title != "Rappel : Votre trajet est-il parti avec des passagers ?" || n.sent[0].InboxID != "f1" {
		t.Errorf("unexpected pushes: %+v", n.sent)
	}
	if len(n.recorded) != 0 {
		t.Errorf("reminders must not create inbox items")
	}
}

func TestSendInboxReminders_QuietHoursSendNothing(t *testing.T) {
	h := time.Now().UTC().Hour()
	quiet := usecase.QuietHours{Start: h, End: (h + 1) % 24} // the current hour
	inbox := &mockInboxRepo{due: []domain.InboxItem{{ID: "n1", Phone: "0611", Kind: domain.KindDriverShared}}}
	n := &mockNotifier{}
	uc := usecase.NewSendInboxReminders(inbox, &mockSubRepo{subs: map[string]domain.Subscription{}}, n, 3, quiet, time.UTC)
	if err := uc.Execute(); err != nil {
		t.Fatal(err)
	}
	if inbox.dueCalled || len(n.sent) != 0 || len(inbox.reminded) != 0 {
		t.Errorf("quiet hours: expected no work, got query=%v sent=%d reminded=%v", inbox.dueCalled, len(n.sent), inbox.reminded)
	}
}

func TestSendInboxReminders_DefaultsInterval(t *testing.T) {
	inbox := &mockInboxRepo{}
	before := time.Now()
	_ = usecase.NewSendInboxReminders(inbox, &mockSubRepo{}, &mockNotifier{}, 0, usecase.QuietHours{}, nil).Execute()
	want := before.Add(-time.Duration(usecase.DefaultInboxReminderIntervalHours) * time.Hour)
	if d := inbox.dueCutoff.Sub(want); d < 0 || d > time.Minute {
		t.Errorf("cutoff %v, want ≈ %v", inbox.dueCutoff, want)
	}
}

func TestSendInboxReminders_PropagatesQueryError(t *testing.T) {
	uc := usecase.NewSendInboxReminders(&failingDueRepo{}, &mockSubRepo{}, &mockNotifier{}, 3, usecase.QuietHours{}, time.UTC)
	if err := uc.Execute(); err == nil {
		t.Error("expected the repository error")
	}
}

type failingDueRepo struct{ mockInboxRepo }

func (f *failingDueRepo) FindRemindersDue([]string, int, time.Time, int) ([]domain.InboxItem, error) {
	return nil, errors.New("db down")
}

// ── The original push carries the inbox id; acting on a request acknowledges ──

func TestSendToAll_PushCarriesInboxID(t *testing.T) {
	n := &mockNotifier{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{"0611": {Phone: "0611", Endpoint: "https://push.example.com"}}}
	usecase.NotifySearcher(domain.Request{ID: "req-1", Phone: "0611"}, domain.Ride{ID: "r1", DriverName: "Bob"}, subs, n)
	if len(n.sent) != 1 || n.sent[0].InboxID != "inbox-1" {
		t.Errorf("push should carry the recorded inbox id, got %+v", n.sent)
	}
}

func TestAcceptInterest_AcknowledgesTheRequest(t *testing.T) {
	interest := domain.Interest{ID: "int-1", RideID: "ride-1", SearcherPhone: "555-searcher", Status: "pending"}
	interests := &mockInterestRepo{byID: map[string]domain.Interest{"int-1": interest}, saved: []domain.Interest{interest}}
	rides := &mockRideRepo{byID: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-driver"}}}
	n := &mockNotifier{}
	if _, _, err := usecase.NewAcceptInterest(interests, rides, &mockSubRepo{subs: map[string]domain.Subscription{}}, n).Execute("int-1", "555-driver"); err != nil {
		t.Fatal(err)
	}
	if len(n.acked) != 1 || n.acked[0] != "555-driver/interest_received:int-1" {
		t.Errorf("acked = %v", n.acked)
	}
}

func TestCancelInterest_AcknowledgesTheDriversNotification(t *testing.T) {
	interest := domain.Interest{ID: "int-1", RideID: "ride-1", SearcherPhone: "555-searcher", SearcherName: "Bob", Status: "pending"}
	interests := &mockInterestRepo{byID: map[string]domain.Interest{"int-1": interest}}
	rides, subs, n := cancelDeps("555-driver")
	if err := usecase.NewCancelInterest(interests, rides, subs, n).Execute("int-1", "555-searcher"); err != nil {
		t.Fatal(err)
	}
	if len(n.acked) != 1 || n.acked[0] != "555-driver/interest_received:int-1" {
		t.Errorf("acked = %v", n.acked)
	}
}

func TestInboxNotifier_AcknowledgeMarksByKey(t *testing.T) {
	inbox := &mockInboxRepo{}
	usecase.NewInboxNotifier(&mockNotifier{}, inbox).Acknowledge("0611", "interest_received:i1")
	if len(inbox.ackedKeys) != 1 || inbox.ackedKeys[0] != "0611/interest_received:i1" {
		t.Errorf("ackedKeys = %v", inbox.ackedKeys)
	}
}

// ── Ride matches: pushed once, reminded by the inbox, stopped by acting ───────

func TestNotifySearcher_LinksTheMatchToRideAndRequest(t *testing.T) {
	n := &mockNotifier{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	usecase.NotifySearcher(domain.Request{ID: "req-7", Phone: "0611"}, domain.Ride{ID: "r1", DriverName: "Bob"}, subs, n)
	assertInbox(t, n, "0611", domain.Message{
		Kind: domain.KindRideMatch, DedupKey: "ride_match:r1", ContactName: "Bob", RideID: "r1", RequestID: "req-7",
	})
}

func TestExpressInterest_AcknowledgesTheMatch(t *testing.T) {
	rides := &mockRideRepo{byID: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-driver"}}}
	n := &mockNotifier{}
	if _, err := usecase.NewExpressInterest(rides, &mockInterestRepo{}, &mockSubRepo{subs: map[string]domain.Subscription{}}, n).Execute("ride-1", "555-searcher", "Bob"); err != nil {
		t.Fatal(err)
	}
	if len(n.acked) != 1 || n.acked[0] != "555-searcher/ride_match:ride-1" {
		t.Errorf("acked = %v", n.acked)
	}
}

func TestPingSearcher_AcknowledgesTheMatch(t *testing.T) {
	reqs := &mockRequestRepoByID{byID: map[string]domain.Request{"req-1": {ID: "req-1", Phone: "555-searcher"}}}
	rides := &mockRideRepo{byID: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-driver", DriverName: "Bob"}}}
	n := &mockNotifier{}
	if _, err := usecase.NewPingSearcher(reqs, rides, &mockInterestRepo{}, &mockSubRepo{subs: map[string]domain.Subscription{}}, n).Execute("req-1", "ride-1", "555-driver"); err != nil {
		t.Fatal(err)
	}
	if len(n.acked) != 1 || n.acked[0] != "555-searcher/ride_match:ride-1" {
		t.Errorf("acked = %v", n.acked)
	}
}
