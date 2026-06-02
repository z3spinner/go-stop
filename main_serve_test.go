package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// buildSPARouter mirrors the NoRoute SPA fallback wiring in main.go for testing.
func buildSPARouter(buildDir string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/ping", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		clean := filepath.Clean(p)
		file := filepath.Join(buildDir, clean)
		if strings.HasPrefix(file, filepath.Clean(buildDir)+string(os.PathSeparator)) {
			if fi, err := os.Stat(file); err == nil && !fi.IsDir() {
				c.File(file)
				return
			}
		}
		c.File(filepath.Join(buildDir, "index.html"))
	})
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
}
