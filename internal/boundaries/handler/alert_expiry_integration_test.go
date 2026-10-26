// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package handler_test

import (
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

func renew(r *gin.Engine, id, phone string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/requests/"+id+"/renew", nil)
	if phone != "" {
		req.Header.Set("X-Phone", phone)
	}
	r.ServeHTTP(w, req)
	return w
}

func postAlert(t *testing.T, r *gin.Engine, body map[string]interface{}) domain.Request {
	t.Helper()
	w := postJSON(r, "/api/requests", body)
	if w.Code/100 != 2 {
		t.Fatalf("post request: %d %s", w.Code, w.Body.String())
	}
	var req domain.Request
	if err := json.Unmarshal(w.Body.Bytes(), &req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return req
}

// The whole cycle: a new anytime alert lasts 30 days; three days before it
// expires its owner finds a warning in the inbox; renewing extends it by 30 days
// from now and marks the warning read; it isn't warned about again until its new
// expiry comes close.
func TestHTTP_AlertExpiry_WarnThenRenew(t *testing.T) {
	truncateAll(t)
	r := setupRouter()
	const phone = "5596001"

	alert := postAlert(t, r, map[string]interface{}{
		"searcher_name": "Zoé", "phone": phone, "origin": "Crest", "destination": "Saillans",
	})
	if d := alert.ExpiresAt.Sub(time.Now().Add(domain.AlertLifetime)); d.Abs() > time.Minute {
		t.Errorf("new anytime alert expires %v, want ≈ 30 days from now", alert.ExpiresAt)
	}

	// Two days left: due for a warning.
	if _, err := handlerPool.Exec(context.Background(),
		`UPDATE requests SET expires_at = NOW() + INTERVAL '2 days' WHERE id = $1`, alert.ID); err != nil {
		t.Fatal(err)
	}
	reqRepo := postgres.NewRequestRepo(handlerPool, 60, time.UTC)
	n := usecase.NewInboxNotifier(&noopNotifier{}, postgres.NewInboxRepo(handlerPool))
	warn := usecase.NewSendAlertExpiryWarnings(reqRepo, postgres.NewSubscriptionRepo(handlerPool), n, usecase.QuietHours{}, time.UTC)
	if err := warn.Execute(); err != nil {
		t.Fatalf("warnings: %v", err)
	}
	if err := warn.Execute(); err != nil { // the next hourly run adds nothing
		t.Fatalf("warnings: %v", err)
	}
	inbox := getInbox(t, r, phone)
	if len(inbox.Items) != 1 || inbox.Items[0].Kind != domain.KindAlertExpiring || inbox.Items[0].Read {
		t.Fatalf("expected one unread alert_expiring item, got %+v", inbox.Items)
	}
	if inbox.Items[0].URL != "/my-searches?alert="+alert.ID || inbox.Items[0].Origin != "Crest" {
		t.Errorf("unexpected item: %+v", inbox.Items[0])
	}

	w := renew(r, alert.ID, phone)
	if w.Code != http.StatusOK {
		t.Fatalf("renew: %d %s", w.Code, w.Body.String())
	}
	var renewed domain.Request
	_ = json.Unmarshal(w.Body.Bytes(), &renewed)
	if d := renewed.ExpiresAt.Sub(time.Now().Add(domain.AlertLifetime)); d.Abs() > time.Minute {
		t.Errorf("renewed alert expires %v, want ≈ 30 days from now", renewed.ExpiresAt)
	}
	if got := getInbox(t, r, phone); got.Unread != 0 {
		t.Errorf("renewing should mark the warning read, unread = %d", got.Unread)
	}
	if err := warn.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := getInbox(t, r, phone); len(got.Items) != 1 {
		t.Errorf("no new warning right after renewing, got %d items", len(got.Items))
	}
}

func TestHTTP_AlertRenew_Rejections(t *testing.T) {
	truncateAll(t)
	r := setupRouter()
	anytime := postAlert(t, r, map[string]interface{}{
		"searcher_name": "Zoé", "phone": "5596002", "origin": "Crest", "destination": "Saillans",
	})
	dated := postAlert(t, r, map[string]interface{}{
		"searcher_name": "Zoé", "phone": "5596002", "origin": "Crest", "destination": "Saillans",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	cases := []struct {
		name, id, phone string
		want            int
	}{
		{"no phone", anytime.ID, "", http.StatusUnauthorized},
		{"someone else", anytime.ID, "5596999", http.StatusForbidden},
		{"unknown", "00000000-0000-0000-0000-000000000000", "5596002", http.StatusNotFound},
		{"dated request", dated.ID, "5596002", http.StatusConflict},
	}
	for _, tc := range cases {
		if w := renew(r, tc.id, tc.phone); w.Code != tc.want {
			t.Errorf("%s: got %d %s, want %d", tc.name, w.Code, w.Body.String(), tc.want)
		}
	}
}
