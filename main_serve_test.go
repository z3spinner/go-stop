// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// buildSPARouter wires the real spaHandler for testing (no ride lookup).
func buildSPARouter(buildDir string) *gin.Engine {
	return buildSPARouterWith(buildDir, "Go-Stop", nil)
}

func buildSPARouterWith(buildDir, siteName string, lookup rideLookupFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/ping", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.NoRoute(spaHandler(buildDir, siteName, lookup, time.UTC))
	return r
}

func TestSPAFallbackServesIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>INDEX"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := buildSPARouter(dir)

	// Deep link with no matching file → index.html
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/my-rides", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "INDEX") {
		t.Fatalf("deep link: got %d %q", w.Code, w.Body.String())
	}

	// Unknown API route → JSON 404, never index.html
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/nope", nil))
	if w.Code != 404 || strings.Contains(w.Body.String(), "INDEX") {
		t.Fatalf("api 404: got %d %q", w.Code, w.Body.String())
	}

	// Existing file is served directly, not replaced by index.html
	if err := os.WriteFile(filepath.Join(dir, "sw.js"), []byte("SWJS"), 0o644); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/sw.js", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "SWJS") || strings.Contains(w.Body.String(), "INDEX") {
		t.Fatalf("existing file: got %d %q", w.Code, w.Body.String())
	}

	// Path traversal attempt is contained — Gin rejects it (400) or falls back to
	// index.html; either way the response must NOT serve arbitrary system file content.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/../../etc/passwd", nil))
	body := w.Body.String()
	if strings.Contains(body, "root:") {
		t.Fatalf("traversal: response looks like /etc/passwd — got %d %q", w.Code, body)
	}
	// Must be either a rejection (400) or the SPA shell (200 with INDEX).
	if w.Code != 400 && !(w.Code == 200 && strings.Contains(body, "INDEX")) {
		t.Fatalf("traversal: unexpected response %d %q", w.Code, body)
	}
}

const shellWithHead = "<!doctype html><html><head><meta charset=\"utf-8\"></head><body>APP</body></html>"

func TestOGInjection_RideRouteAndDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(shellWithHead), 0o644); err != nil {
		t.Fatal(err)
	}
	lookup := func(id string) (ogRide, bool) {
		if id != "ride-123" {
			return ogRide{}, false
		}
		return ogRide{
			Origin: "Saillans", Destination: "Crest", Flexibility: 30,
			DepartureAt: time.Date(2030, 6, 3, 14, 30, 0, 0, time.UTC),
		}, true
	}
	r := buildSPARouterWith(dir, "Go Stop Saillans!", lookup)

	// A ride page gets the route as og:title and the departure as og:description.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/rides/ride-123", nil))
	body := w.Body.String()
	for _, want := range []string{
		`property="og:title" content="Saillans → Crest"`,
		`lun. 3 juin à 14:30 · ±30 min`, // 2030-06-03 is a Monday
		`property="og:image"`,
		`name="twitter:card" content="summary_large_image"`,
		"<body>APP</body>", // original shell preserved
	} {
		if !strings.Contains(body, want) {
			t.Errorf("ride OG missing %q in:\n%s", want, body)
		}
	}

	// An unknown ride id falls back to the site default (no route leaked).
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/rides/missing", nil))
	if b := w.Body.String(); !strings.Contains(b, `property="og:title" content="Go Stop Saillans!"`) {
		t.Errorf("unknown ride should use the default og:title, got:\n%s", b)
	}

	// The home page gets the site default title + description.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	b := w.Body.String()
	if !strings.Contains(b, `property="og:title" content="Go Stop Saillans!"`) ||
		!strings.Contains(b, `content="Trajets locaux, contact direct"`) {
		t.Errorf("home OG missing default tags, got:\n%s", b)
	}
}

func TestOGInjection_EscapesAndAbsoluteURLs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(shellWithHead), 0o644); err != nil {
		t.Fatal(err)
	}
	lookup := func(id string) (ogRide, bool) {
		return ogRide{Origin: `A<b>"x"`, Destination: "Crest", Flexibility: 0,
			DepartureAt: time.Date(2030, 1, 2, 8, 5, 0, 0, time.UTC)}, true
	}
	r := buildSPARouterWith(dir, "Go-Stop", lookup)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/rides/x", nil)
	req.Host = "go-stop.example"
	req.Header.Set("X-Forwarded-Proto", "https")
	r.ServeHTTP(w, req)
	body := w.Body.String()

	if strings.Contains(body, `A<b>"x"`) {
		t.Errorf("dynamic value not HTML-escaped:\n%s", body)
	}
	if !strings.Contains(body, `content="https://go-stop.example/rides/x"`) {
		t.Errorf("og:url should be absolute from forwarded proto + host, got:\n%s", body)
	}
	if !strings.Contains(body, `content="https://go-stop.example/og-image.png"`) {
		t.Errorf("og:image should be an absolute URL, got:\n%s", body)
	}
}
