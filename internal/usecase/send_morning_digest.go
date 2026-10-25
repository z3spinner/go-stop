// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// DefaultMorningDigestHour is the local hour (SERVICE_TZ) the digest goes out.
const DefaultMorningDigestHour = 8

// morningDigestWindow: a run up to this many hours after the send hour (a
// restart, a late cron tick) still sends that day's digest; later it's skipped.
const morningDigestWindow = 4

// morningDigestSettingPrefix + "YYYY-MM-DD" marks a day's digest as sent.
const morningDigestSettingPrefix = "morning_digest:"

// morningDigestMaxLines caps the trips named in the push; the rest are counted.
const morningDigestMaxLines = 2

// SendMorningDigest sends, once a day at the configured hour, a summary of the
// day's board to every phone with notifications on that hasn't turned the
// digest off: first what concerns that person (rides matching their alerts,
// people looking for a trip they drive today), then how many rides and requests
// there are today. Nothing is sent to someone for whom today has nothing but
// their own posts, and nothing at all on an empty day. Push text is French, like
// the other server-initiated pushes; the inbox renders the title in the reader's
// language. One inbox item per phone per day, kept a day.
type SendMorningDigest struct {
	rides    repository.RideRepository
	requests repository.RequestRepository
	subs     repository.SubscriptionRepository
	prefs    repository.NotificationPreferenceRepository
	settings repository.SettingsRepository
	notifier notification.Notifier
	loc      *time.Location
	hour     int // < 0: disabled
	siteName string
	// Clock is the time source; defaults to time.Now. Override in tests.
	Clock func() time.Time
}

func NewSendMorningDigest(
	rides repository.RideRepository,
	requests repository.RequestRepository,
	subs repository.SubscriptionRepository,
	prefs repository.NotificationPreferenceRepository,
	settings repository.SettingsRepository,
	notifier notification.Notifier,
	loc *time.Location,
	hour int,
	siteName string,
) *SendMorningDigest {
	if loc == nil {
		loc = time.UTC
	}
	if siteName == "" {
		siteName = "Go-Stop"
	}
	return &SendMorningDigest{
		rides: rides, requests: requests, subs: subs, prefs: prefs, settings: settings,
		notifier: notifier, loc: loc, hour: hour, siteName: siteName, Clock: time.Now,
	}
}

// ParseMorningDigestHour reads MORNING_DIGEST_HOUR: "" → the default, 0–23 →
// that hour, "none"/"off" → disabled (-1).
func ParseMorningDigestHour(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "":
		return DefaultMorningDigestHour, nil
	case "none", "off":
		return -1, nil
	}
	h, err := strconv.Atoi(s)
	if err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("invalid MORNING_DIGEST_HOUR %q (want 0-23 or none)", s)
	}
	return h, nil
}

func (uc *SendMorningDigest) Execute() error {
	if uc.hour < 0 {
		return nil
	}
	now := uc.Clock().In(uc.loc)
	if now.Hour() < uc.hour || now.Hour() >= uc.hour+morningDigestWindow {
		return nil
	}
	day := now.Format("2006-01-02")
	key := morningDigestSettingPrefix + day
	ctx := context.Background()
	if _, sent, err := uc.settings.Get(ctx, key); err != nil {
		return fmt.Errorf("morning digest: check sent: %w", err)
	} else if sent {
		return nil
	}

	allRides, err := uc.rides.FindAll()
	if err != nil {
		return fmt.Errorf("morning digest: rides: %w", err)
	}
	allRequests, err := uc.requests.FindAllActive()
	if err != nil {
		return fmt.Errorf("morning digest: requests: %w", err)
	}
	rides, requests := todaysBoard(allRides, allRequests, now, uc.loc)
	if len(rides) == 0 && len(requests) == 0 {
		log.Printf("morning digest: nothing on the board today, no push")
		return uc.settings.InsertIfAbsent(ctx, key, "sent")
	}

	// What concerns each person: rides matching their alerts, and people whose
	// alerts match a ride they drive today.
	personal := map[string][]string{}
	seen := map[string]bool{}
	add := func(phone, id, line string) {
		if phone == "" || seen[phone+"|"+id] {
			return
		}
		seen[phone+"|"+id] = true
		personal[phone] = append(personal[phone], line)
	}
	for _, ride := range rides {
		matches, err := uc.requests.FindMatching(ride)
		if err != nil {
			log.Printf("morning digest: matching ride %s: %v", ride.ID, err)
			continue
		}
		for _, req := range matches {
			if req.Phone == ride.Phone {
				continue
			}
			add(req.Phone, "ride:"+ride.ID, digestRideLine(ride, uc.loc))
			add(ride.Phone, "request:"+req.ID, digestRequestLine(req, uc.loc))
		}
	}

	phones, err := uc.recipients()
	if err != nil {
		return err
	}
	sent := 0
	for _, phone := range phones {
		body, ok := digestBody(phone, personal[phone], rides, requests, uc.loc)
		if !ok {
			continue
		}
		sendToAll(phone, domain.Message{
			Title:    "☀️ Aujourd'hui sur " + uc.siteName,
			Body:     body,
			URL:      "/",
			Kind:     domain.KindMorningDigest,
			DedupKey: domain.InboxKey(domain.KindMorningDigest, day),
		}, uc.subs, uc.notifier)
		sent++
	}
	log.Printf("morning digest: %d rides, %d requests today; sent to %d of %d phones", len(rides), len(requests), sent, len(phones))
	return uc.settings.InsertIfAbsent(ctx, key, "sent")
}

// recipients: every phone with a push device, minus those that turned the
// digest off. (Devices without a phone have no preferences to honour and no
// alerts to personalise with, so they don't get it.)
func (uc *SendMorningDigest) recipients() ([]string, error) {
	subs, err := uc.subs.FindAll()
	if err != nil {
		return nil, fmt.Errorf("morning digest: subscribers: %w", err)
	}
	optOuts, err := uc.prefs.MorningDigestOptOuts()
	if err != nil {
		return nil, fmt.Errorf("morning digest: preferences: %w", err)
	}
	off := map[string]bool{}
	for _, p := range optOuts {
		off[p] = true
	}
	set := map[string]bool{}
	for _, s := range subs {
		if s.Phone != "" && !off[s.Phone] {
			set[s.Phone] = true
		}
	}
	phones := make([]string, 0, len(set))
	for p := range set {
		phones = append(phones, p)
	}
	sort.Strings(phones)
	return phones, nil
}

// todaysBoard keeps the rides leaving later today and the requests for today
// (a date, with or without a time) that haven't passed yet. Open-ended alerts
// (anytime, daily) aren't about today, so they only count through matches.
func todaysBoard(rides []domain.Ride, requests []domain.Request, now time.Time, loc *time.Location) ([]domain.Ride, []domain.Request) {
	today := now.Format("2006-01-02")
	var outRides []domain.Ride
	for _, r := range rides {
		dep := r.DepartureAt.In(loc)
		if dep.Format("2006-01-02") == today && !dep.Add(time.Duration(r.Flexibility)*time.Minute).Before(now) {
			outRides = append(outRides, r)
		}
	}
	sort.Slice(outRides, func(i, j int) bool { return outRides[i].DepartureAt.Before(outRides[j].DepartureAt) })
	var outReqs []domain.Request
	for _, r := range requests {
		if r.Date.IsZero() || r.Date.Format("2006-01-02") != today {
			continue
		}
		if hasClockTime(r) && r.DepartureAt.Add(time.Duration(r.Flexibility)*time.Minute).Before(now) {
			continue
		}
		outReqs = append(outReqs, r)
	}
	return outRides, outReqs
}

// hasClockTime: the request is for a given time (not a whole day, and not the
// 1970 daily sentinel).
func hasClockTime(r domain.Request) bool {
	return !r.DepartureAt.IsZero() && r.DepartureAt.Year() != 1970
}

// digestBody is the push text for one phone, or false when today has nothing
// for them: no match, and nothing on the board but their own posts.
func digestBody(phone string, mine []string, rides []domain.Ride, requests []domain.Request, loc *time.Location) (string, bool) {
	var others []string // other people's posts today, earliest first
	nRides, nReqs := 0, 0
	for _, r := range rides {
		if r.Phone != phone {
			nRides++
			others = append(others, digestRideLine(r, loc))
		}
	}
	for _, r := range requests {
		if r.Phone != phone {
			nReqs++
			others = append(others, digestRequestLine(r, loc))
		}
	}
	if len(mine) == 0 && nRides+nReqs == 0 {
		return "", false
	}
	var b strings.Builder
	if len(mine) > 0 {
		b.WriteString("Pour vous : ")
		b.WriteString(joinLines(mine))
		b.WriteString(".")
		if counts := digestCounts(nRides, nReqs); counts != "" {
			b.WriteString(" Aujourd'hui : " + counts + ".")
		}
		return b.String(), true
	}
	b.WriteString("Aujourd'hui : " + digestCounts(nRides, nReqs) + " — ")
	b.WriteString(joinLines(others))
	b.WriteString(".")
	return b.String(), true
}

func joinLines(lines []string) string {
	if len(lines) <= morningDigestMaxLines {
		return strings.Join(lines, " · ")
	}
	return fmt.Sprintf("%s (+%d)", strings.Join(lines[:morningDigestMaxLines], " · "), len(lines)-morningDigestMaxLines)
}

// digestCounts: "3 trajets, 1 demande" (zero parts left out).
func digestCounts(nRides, nReqs int) string {
	var parts []string
	if nRides > 0 {
		parts = append(parts, plural(nRides, "trajet", "trajets"))
	}
	if nReqs > 0 {
		parts = append(parts, plural(nReqs, "demande", "demandes"))
	}
	return strings.Join(parts, ", ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// "Saillans → Crest à 7h15 (Alice)"
func digestRideLine(r domain.Ride, loc *time.Location) string {
	line := fmt.Sprintf("%s → %s à %s", r.Origin, r.Destination, frenchClock(r.DepartureAt.In(loc)))
	if name := strings.TrimSpace(r.DriverName); name != "" {
		line += " (" + name + ")"
	}
	return line
}

// "Marie cherche Valence TGV → Saillans à 18h45" (no time for a whole day or
// an open-ended alert; a daily alert's time is its usual time).
func digestRequestLine(r domain.Request, loc *time.Location) string {
	who := strings.TrimSpace(r.SearcherName)
	if who == "" {
		who = "Quelqu'un"
	}
	line := fmt.Sprintf("%s cherche %s → %s", who, r.Origin, r.Destination)
	switch {
	case hasClockTime(r):
		line += " à " + frenchClock(r.DepartureAt.In(loc))
	case !r.DepartureAt.IsZero(): // daily: the time is stored on 1970-01-01 UTC
		line += " vers " + frenchClock(r.DepartureAt)
	}
	return line
}

// "7h15", "18h00"
func frenchClock(t time.Time) string {
	return fmt.Sprintf("%dh%02d", t.Hour(), t.Minute())
}
