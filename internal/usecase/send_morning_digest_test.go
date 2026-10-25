// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

// digestRequestRepo matches a ride to the active requests on the same route
// (the real matching is tested against Postgres).
type digestRequestRepo struct {
	active []domain.Request
}

func (m *digestRequestRepo) Save(domain.Request) error { return nil }
func (m *digestRequestRepo) FindByID(string) (domain.Request, error) {
	return domain.Request{}, errors.New("not found")
}
func (m *digestRequestRepo) FindByPhone(string) ([]domain.Request, error) { return nil, nil }
func (m *digestRequestRepo) FindAllActive() ([]domain.Request, error)     { return m.active, nil }
func (m *digestRequestRepo) Delete(string) error                          { return nil }
func (m *digestRequestRepo) DeleteExpired() error                         { return nil }
func (m *digestRequestRepo) FindMatching(ride domain.Ride) ([]domain.Request, error) {
	var out []domain.Request
	for _, r := range m.active {
		if r.Origin == ride.Origin && r.Destination == ride.Destination {
			out = append(out, r)
		}
	}
	return out, nil
}

type mockPrefsRepo struct {
	optOuts []string
	set     map[string]bool
}

func (m *mockPrefsRepo) Get(phone string) (domain.NotificationPreferences, error) {
	if on, ok := m.set[phone]; ok {
		return domain.NotificationPreferences{MorningDigest: on}, nil
	}
	return domain.DefaultNotificationPreferences(), nil
}
func (m *mockPrefsRepo) SetMorningDigest(phone string, on bool) error {
	if m.set == nil {
		m.set = map[string]bool{}
	}
	m.set[phone] = on
	return nil
}
func (m *mockPrefsRepo) MorningDigestOptOuts() ([]string, error) { return m.optOuts, nil }

// ── fixtures ──────────────────────────────────────────────────────────────────

var paris, _ = time.LoadLocation("Europe/Paris")

// Monday 26 Oct 2026, 8:05 in Saillans.
var digestNow = time.Date(2026, 10, 26, 8, 5, 0, 0, paris)

func at(h, m int) time.Time { return time.Date(2026, 10, 26, h, m, 0, 0, paris) }

func subsFor(phones ...string) *mockSubRepo {
	s := &mockSubRepo{subs: map[string]domain.Subscription{}}
	for _, p := range phones {
		s.subs[p] = domain.Subscription{Phone: p, Endpoint: "https://push.example.com/" + p}
	}
	return s
}

type digestFixture struct {
	rides    *mockRideRepo
	requests *digestRequestRepo
	subs     *mockSubRepo
	prefs    *mockPrefsRepo
	settings *mockSettingsRepo
	n        *mockNotifier
}

func (f *digestFixture) run(t *testing.T, now time.Time, hour int) {
	t.Helper()
	uc := usecase.NewSendMorningDigest(f.rides, f.requests, f.subs, f.prefs, f.settings, f.n, paris, hour, "Go-Stop")
	uc.Clock = func() time.Time { return now }
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// bodies: the recorded digest per phone.
func (f *digestFixture) bodies() map[string]domain.Message {
	out := map[string]domain.Message{}
	for _, r := range f.n.recorded {
		out[r.phone] = r.msg
	}
	return out
}

func newDigestFixture() *digestFixture {
	return &digestFixture{
		rides: &mockRideRepo{saved: []domain.Ride{
			{ID: "ride-am", DriverName: "Alice", Phone: "06-alice", Origin: "Saillans", Destination: "Crest", DepartureAt: at(17, 50)},
			{ID: "ride-early", DriverName: "Bruno", Phone: "06-bruno", Origin: "Saillans", Destination: "Die", DepartureAt: at(7, 15)},                          // already gone
			{ID: "ride-tomorrow", DriverName: "Bruno", Phone: "06-bruno", Origin: "Crest", Destination: "Saillans", DepartureAt: at(7, 15).Add(24 * time.Hour)}, // not today
		}},
		requests: &digestRequestRepo{active: []domain.Request{
			// Zoé's anytime alert matches Alice's ride.
			{ID: "req-zoe", SearcherName: "Zoé", Phone: "06-zoe", Origin: "Saillans", Destination: "Crest"},
			// Marie needs a lift from Valence tonight: on today's board.
			{ID: "req-marie", SearcherName: "Marie", Phone: "06-marie", Origin: "Valence TGV", Destination: "Saillans",
				Date: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC), DepartureAt: at(18, 45), Flexibility: 30},
		}},
		subs:     subsFor("06-alice", "06-zoe", "06-marie", "06-bystander", "06-optout"),
		prefs:    &mockPrefsRepo{optOuts: []string{"06-optout"}},
		settings: newMockSettings(),
		n:        &mockNotifier{},
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestSendMorningDigest_PersonalFirstThenTheBoard(t *testing.T) {
	f := newDigestFixture()
	f.run(t, digestNow, 8)
	got := f.bodies()

	// Zoé: Alice's ride matches her alert.
	if b := got["06-zoe"].Body; b != "Pour vous : Saillans → Crest à 17h50 (Alice). Aujourd'hui : 1 trajet, 1 demande." {
		t.Errorf("zoé body = %q", b)
	}
	// Alice: someone is looking for the trip she drives today; her own ride isn't counted.
	if b := got["06-alice"].Body; b != "Pour vous : Zoé cherche Saillans → Crest. Aujourd'hui : 1 demande." {
		t.Errorf("alice body = %q", b)
	}
	// Marie: nothing personal, the board without her own request.
	if b := got["06-marie"].Body; b != "Aujourd'hui : 1 trajet — Saillans → Crest à 17h50 (Alice)." {
		t.Errorf("marie body = %q", b)
	}
	// Someone with no posts: the whole board, earliest rides first.
	if b := got["06-bystander"].Body; b != "Aujourd'hui : 1 trajet, 1 demande — Saillans → Crest à 17h50 (Alice) · Marie cherche Valence TGV → Saillans à 18h45." {
		t.Errorf("bystander body = %q", b)
	}
	if _, ok := got["06-optout"]; ok {
		t.Error("a phone that turned the digest off must not get it")
	}
	if len(got) != 4 {
		t.Errorf("expected 4 digests, got %d: %v", len(got), got)
	}
	msg := got["06-bystander"]
	if msg.Kind != domain.KindMorningDigest || msg.DedupKey != "morning_digest:2026-10-26" || msg.URL != "/" || msg.Title != "☀️ Aujourd'hui sur Go-Stop" {
		t.Errorf("unexpected message: %+v", msg)
	}
	if !f.n.called {
		t.Error("expected pushes to devices")
	}
	if len(f.settings.insertCalled) != 1 || f.settings.insertCalled[0] != "morning_digest:2026-10-26" {
		t.Errorf("expected the day marked sent, got %v", f.settings.insertCalled)
	}
}

func TestSendMorningDigest_OncePerDay(t *testing.T) {
	f := newDigestFixture()
	f.run(t, digestNow, 8)
	first := len(f.n.recorded)
	f.run(t, digestNow.Add(time.Hour), 8)
	if len(f.n.recorded) != first {
		t.Errorf("second run the same day sent again: %d → %d", first, len(f.n.recorded))
	}
}

func TestSendMorningDigest_OnlyAroundTheHour(t *testing.T) {
	for _, now := range []time.Time{at(7, 59), at(12, 0), at(22, 0)} {
		f := newDigestFixture()
		f.run(t, now, 8)
		if len(f.n.recorded) != 0 || len(f.settings.insertCalled) != 0 {
			t.Errorf("%s: nothing should happen outside 8:00–11:59", now.Format("15:04"))
		}
	}
	// A late run within the window (server restart) still sends.
	f := newDigestFixture()
	f.run(t, at(11, 30), 8)
	if len(f.n.recorded) == 0 {
		t.Error("11:30: the day's digest should still go out")
	}
}

func TestSendMorningDigest_Disabled(t *testing.T) {
	f := newDigestFixture()
	f.run(t, digestNow, -1)
	if len(f.n.recorded) != 0 || len(f.settings.insertCalled) != 0 {
		t.Error("disabled: nothing should happen")
	}
}

func TestSendMorningDigest_EmptyDaySendsNothing(t *testing.T) {
	f := newDigestFixture()
	f.rides.saved = nil
	f.requests.active = []domain.Request{{ID: "req-zoe", Phone: "06-zoe", Origin: "Saillans", Destination: "Crest"}} // an open-ended alert isn't today's board
	f.run(t, digestNow, 8)
	if len(f.n.recorded) != 0 {
		t.Errorf("empty day: nothing to send, got %d", len(f.n.recorded))
	}
	if len(f.settings.insertCalled) != 1 {
		t.Error("the empty day is still marked done")
	}
}

func TestSendMorningDigest_OwnPostsOnlyIsNothing(t *testing.T) {
	f := newDigestFixture()
	f.requests.active = nil // Alice's ride alone on the board
	f.run(t, digestNow, 8)
	if _, ok := f.bodies()["06-alice"]; ok {
		t.Error("a board with only your own ride says nothing new to you")
	}
	if b := f.bodies()["06-bystander"].Body; !strings.HasPrefix(b, "Aujourd'hui : 1 trajet — ") {
		t.Errorf("others still get it, got %q", b)
	}
}

func TestSendMorningDigest_CapsTheNamedTrips(t *testing.T) {
	f := newDigestFixture()
	f.rides.saved = nil
	for i, h := range []int{9, 10, 11, 12} {
		f.rides.saved = append(f.rides.saved, domain.Ride{ID: string(rune('a' + i)), DriverName: "D", Phone: "06-d", Origin: "A", Destination: "B", DepartureAt: at(h, 0)})
	}
	f.requests.active = nil
	f.run(t, digestNow, 8)
	if b := f.bodies()["06-bystander"].Body; b != "Aujourd'hui : 4 trajets — A → B à 9h00 (D) · A → B à 10h00 (D) (+2)." {
		t.Errorf("body = %q", b)
	}
}

func TestParseMorningDigestHour(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"", 8, false}, {"8", 8, false}, {" 7 ", 7, false}, {"0", 0, false}, {"23", 23, false},
		{"none", -1, false}, {"OFF", -1, false}, {"24", 0, true}, {"-1", 0, true}, {"8h", 0, true},
	}
	for _, tc := range cases {
		got, err := usecase.ParseMorningDigestHour(tc.in)
		if (err != nil) != tc.wantErr || (!tc.wantErr && got != tc.want) {
			t.Errorf("ParseMorningDigestHour(%q) = %d, %v; want %d, err=%v", tc.in, got, err, tc.want, tc.wantErr)
		}
	}
}

func TestNotificationPreferences_GetAndSet(t *testing.T) {
	prefs := &mockPrefsRepo{}
	got, err := usecase.NewGetNotificationPreferences(prefs).Execute("06-x")
	if err != nil || !got.MorningDigest {
		t.Fatalf("default should be on, got %+v, %v", got, err)
	}
	if _, err := usecase.NewSetNotificationPreferences(prefs).Execute("06-x", domain.NotificationPreferences{MorningDigest: false}); err != nil {
		t.Fatal(err)
	}
	if got, _ := usecase.NewGetNotificationPreferences(prefs).Execute("06-x"); got.MorningDigest {
		t.Error("expected the digest off after saving")
	}
	if _, err := usecase.NewGetNotificationPreferences(prefs).Execute(""); !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("no phone: err = %v, want ErrUnauthorized", err)
	}
	if _, err := usecase.NewSetNotificationPreferences(prefs).Execute("", domain.NotificationPreferences{}); !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("no phone: err = %v, want ErrUnauthorized", err)
	}
}
