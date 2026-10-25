// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func prefsReq(r *gin.Engine, method, phone string, body interface{}) *httptest.ResponseRecorder {
	if method == http.MethodPut {
		w := httptest.NewRecorder()
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, "/api/notification-preferences", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		if phone != "" {
			req.Header.Set("X-Phone", phone)
		}
		r.ServeHTTP(w, req)
		return w
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, "/api/notification-preferences", nil)
	if phone != "" {
		req.Header.Set("X-Phone", phone)
	}
	r.ServeHTTP(w, req)
	return w
}

func digestPref(t *testing.T, w *httptest.ResponseRecorder) bool {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("preferences: %d %s", w.Code, w.Body.String())
	}
	var p struct {
		MorningDigest bool `json:"morning_digest"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p.MorningDigest
}

func TestHTTP_NotificationPreferences(t *testing.T) {
	truncateAll(t)
	r := setupRouter()
	if !digestPref(t, prefsReq(r, http.MethodGet, "0697000001", nil)) {
		t.Error("the morning digest is on by default")
	}
	if digestPref(t, prefsReq(r, http.MethodPut, "0697000001", map[string]bool{"morning_digest": false})) {
		t.Error("PUT should echo the saved value")
	}
	if digestPref(t, prefsReq(r, http.MethodGet, "0697000001", nil)) {
		t.Error("expected the digest off after saving")
	}
	if !digestPref(t, prefsReq(r, http.MethodGet, "0697000002", nil)) {
		t.Error("other phones keep the default")
	}
	if w := prefsReq(r, http.MethodGet, "", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("no phone: got %d, want 401", w.Code)
	}
	if w := prefsReq(r, http.MethodPut, "", map[string]bool{"morning_digest": true}); w.Code != http.StatusUnauthorized {
		t.Errorf("no phone: got %d, want 401", w.Code)
	}
}

// The digest against the real store and matching: a ride today, an alert that
// matches it, a request for today, and someone who turned the digest off.
func TestHTTP_MorningDigest_EndToEnd(t *testing.T) {
	truncateAll(t)
	// The "sent today" marker lives in app_settings, which truncateAll keeps
	// (it also holds the VAPID keys).
	if _, err := handlerPool.Exec(context.Background(), `DELETE FROM app_settings WHERE key LIKE 'morning_digest:%'`); err != nil {
		t.Fatal(err)
	}
	r := setupRouter()
	paris, _ := time.LoadLocation("Europe/Paris")
	// A day in the future, so everything posted is still active in the store.
	now := time.Date(2030, 6, 3, 8, 5, 0, 0, paris)

	mustPost := func(path string, body map[string]interface{}) {
		t.Helper()
		if w := postJSON(r, path, body); w.Code/100 != 2 {
			t.Fatalf("POST %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	mustPost("/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "0697000010", "origin": "Saillans", "destination": "Crest",
		"departure_at": time.Date(2030, 6, 3, 17, 50, 0, 0, paris).UTC().Format(time.RFC3339), "flexibility": 0,
	})
	mustPost("/api/requests", map[string]interface{}{ // anytime alert matching Alice's ride
		"searcher_name": "Zoé", "phone": "0697000011", "origin": "Saillans", "destination": "Crest",
	})
	mustPost("/api/requests", map[string]interface{}{ // today's request
		"searcher_name": "Marie", "phone": "0697000012", "origin": "Valence TGV", "destination": "Saillans",
		"departure_at": time.Date(2030, 6, 3, 18, 45, 0, 0, paris).UTC().Format(time.RFC3339), "flexibility": 30,
	})

	subRepo := postgres.NewSubscriptionRepo(handlerPool)
	for _, p := range []string{"0697000010", "0697000011", "0697000012", "0697000013", "0697000014"} {
		if err := subRepo.Save(domain.Subscription{Phone: p, Endpoint: "https://push.example.com/" + p, Keys: domain.PushKeys{P256DH: "k", Auth: "a"}}); err != nil {
			t.Fatal(err)
		}
	}
	prefsReq(r, http.MethodPut, "0697000014", map[string]bool{"morning_digest": false})

	n := usecase.NewInboxNotifier(&noopNotifier{}, postgres.NewInboxRepo(handlerPool))
	digest := usecase.NewSendMorningDigest(postgres.NewRideRepo(handlerPool, 60), postgres.NewRequestRepo(handlerPool, 60), subRepo,
		postgres.NewNotificationPreferenceRepo(handlerPool), postgres.NewSettingsRepo(handlerPool), n, paris, 8, "Go-Stop")
	digest.Clock = func() time.Time { return now }
	if err := digest.Execute(); err != nil {
		t.Fatalf("digest: %v", err)
	}
	if err := digest.Execute(); err != nil { // the next hourly run: already sent today
		t.Fatalf("digest: %v", err)
	}

	want := map[string]string{
		"0697000010": "Pour vous : Zoé cherche Saillans → Crest. Aujourd'hui : 1 demande.",
		"0697000011": "Pour vous : Saillans → Crest à 17h50 (Alice). Aujourd'hui : 1 trajet, 1 demande.",
		"0697000012": "Aujourd'hui : 1 trajet — Saillans → Crest à 17h50 (Alice).",
		"0697000013": "Aujourd'hui : 1 trajet, 1 demande — Saillans → Crest à 17h50 (Alice) · Marie cherche Valence TGV → Saillans à 18h45.",
	}
	for phone, body := range want {
		var digests []string
		for _, it := range getInbox(t, r, phone).Items {
			if it.Kind == domain.KindMorningDigest {
				digests = append(digests, it.Body)
			}
		}
		if len(digests) != 1 || digests[0] != body {
			t.Errorf("%s: digests = %q, want [%q]", phone, digests, body)
		}
	}
	for _, it := range getInbox(t, r, "0697000014").Items {
		if it.Kind == domain.KindMorningDigest {
			t.Error("the phone that turned the digest off got one")
		}
	}
}
