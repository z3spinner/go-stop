// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ogRide is the minimal ride data needed to render a link preview.
type ogRide struct {
	Origin      string
	Destination string
	DepartureAt time.Time
	Flexibility int
}

// rideLookupFunc fetches a ride for OG rendering; ok is false when not found.
// Kept as a function (rather than the repository) so spaHandler can be unit
// tested without a database.
type rideLookupFunc func(id string) (ride ogRide, ok bool)

// The preview text is rendered server-side for crawlers, so it can't use the
// client i18n bundle — it's French, matching the app's primary audience.
const ogDefaultDescription = "Trajets locaux, contact direct"

var frWeekdays = [...]string{"dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."}
var frMonths = [...]string{"janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."}

func formatDepartureFR(t time.Time, tz *time.Location) string {
	if tz != nil {
		t = t.In(tz)
	}
	return fmt.Sprintf("%s %d %s à %02d:%02d", frWeekdays[int(t.Weekday())], t.Day(), frMonths[int(t.Month())-1], t.Hour(), t.Minute())
}

func flexLabelFR(flex int) string {
	switch flex {
	case 0:
		return "Exact"
	default:
		return fmt.Sprintf("±%d min", flex)
	}
}

func ogScheme(c *gin.Context) string {
	if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

func ogAbsURL(c *gin.Context, path string) string {
	return ogScheme(c) + "://" + c.Request.Host + path
}

// ridePathID returns the ride id for a /rides/:id path (and only that shape).
func ridePathID(path string) (string, bool) {
	const prefix = "/rides/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(path, prefix)
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// injectOG inserts Open Graph / Twitter meta tags (and a <title>) into the SPA
// shell before </head>, so crawlers that don't run JS still get a rich preview.
// Ride pages (/rides/:id) get the route + departure; every other path gets the
// site default.
func injectOG(shell string, c *gin.Context, siteName string, lookup rideLookupFunc, tz *time.Location) string {
	idx := strings.Index(shell, "</head>")
	if idx < 0 {
		return shell // unexpected shell shape — serve as-is
	}

	title := siteName
	desc := ogDefaultDescription
	if id, ok := ridePathID(c.Request.URL.Path); ok && lookup != nil {
		if r, found := lookup(id); found {
			title = r.Origin + " → " + r.Destination
			desc = formatDepartureFR(r.DepartureAt, tz) + " · " + flexLabelFR(r.Flexibility)
		}
	}

	tags := buildOGTags(title, desc, siteName, ogAbsURL(c, c.Request.URL.Path), ogAbsURL(c, "/og-image.png"))
	return shell[:idx] + tags + shell[idx:]
}

func buildOGTags(title, desc, siteName, pageURL, imageURL string) string {
	e := html.EscapeString
	pageTitle := siteName
	if title != siteName {
		pageTitle = title + " · " + siteName
	}
	var b strings.Builder
	b.WriteString("<title>" + e(pageTitle) + "</title>")
	b.WriteString(`<meta name="description" content="` + e(desc) + `"/>`)
	b.WriteString(`<meta property="og:type" content="website"/>`)
	b.WriteString(`<meta property="og:site_name" content="` + e(siteName) + `"/>`)
	b.WriteString(`<meta property="og:title" content="` + e(title) + `"/>`)
	b.WriteString(`<meta property="og:description" content="` + e(desc) + `"/>`)
	b.WriteString(`<meta property="og:url" content="` + e(pageURL) + `"/>`)
	b.WriteString(`<meta property="og:image" content="` + e(imageURL) + `"/>`)
	b.WriteString(`<meta property="og:image:width" content="1200"/>`)
	b.WriteString(`<meta property="og:image:height" content="630"/>`)
	b.WriteString(`<meta name="twitter:card" content="summary_large_image"/>`)
	b.WriteString(`<meta name="twitter:title" content="` + e(title) + `"/>`)
	b.WriteString(`<meta name="twitter:description" content="` + e(desc) + `"/>`)
	b.WriteString(`<meta name="twitter:image" content="` + e(imageURL) + `"/>`)
	return b.String()
}
