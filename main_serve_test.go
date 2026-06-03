package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// buildSPARouter wires the real spaHandler for testing.
func buildSPARouter(buildDir string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/ping", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.NoRoute(spaHandler(buildDir))
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
