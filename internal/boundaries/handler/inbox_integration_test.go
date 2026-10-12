// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/boundaries/handler"
)

func postWithPhone(r *gin.Engine, path, phone string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Phone", phone)
	r.ServeHTTP(w, req)
	return w
}

func getInbox(t *testing.T, r *gin.Engine, phone string) handler.InboxResponse {
	t.Helper()
	w := getWithPhone(r, "/api/inbox", phone)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /inbox expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "0611000001") {
		t.Errorf("inbox must not expose the other party's phone: %s", w.Body.String())
	}
	var out handler.InboxResponse
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

// An offer reaches the searcher's inbox even though they have no push
// subscription; a repeat offer doesn't duplicate it; read state round-trips.
func TestInbox_OfferContactLandsInSearcherInbox(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	w := postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Bob", "phone": "0622000099",
		"origin": "Saillans", "destination": "Crest",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /requests expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&created)
	requestID := created["ID"].(string)

	offer := map[string]interface{}{"phone": "0611000001", "name": "Alice"}
	for i := 0; i < 2; i++ { // second offer = repeat
		if w := postJSON(r, "/api/requests/"+requestID+"/offer-contact", offer); w.Code != http.StatusNoContent {
			t.Fatalf("offer-contact expected 204, got %d: %s", w.Code, w.Body.String())
		}
	}

	inbox := getInbox(t, r, "06 22 00 00 99") // header is normalized like elsewhere
	if len(inbox.Items) != 1 || inbox.Unread != 1 {
		t.Fatalf("expected 1 unread item, got %d items / %d unread", len(inbox.Items), inbox.Unread)
	}
	it := inbox.Items[0]
	if it.Kind != "contact_offered" || it.ContactName != "Alice" || it.Origin != "Saillans" ||
		it.Destination != "Crest" || it.URL != "/my-searches" || it.Read || it.CreatedAt == "" {
		t.Errorf("unexpected item: %+v", it)
	}

	// Someone else's inbox is empty.
	if other := getInbox(t, r, "0633333333"); len(other.Items) != 0 {
		t.Errorf("expected empty inbox for another phone, got %d items", len(other.Items))
	}
	// Another phone can't mark this item read.
	if w := postWithPhone(r, "/api/inbox/read", "0633333333", map[string]interface{}{"ids": []string{it.ID}}); w.Code != http.StatusNoContent {
		t.Fatalf("mark read expected 204, got %d", w.Code)
	}
	if got := getInbox(t, r, "0622000099"); got.Unread != 1 {
		t.Errorf("another phone must not mark it read; unread=%d", got.Unread)
	}

	// Empty ids without all=true marks nothing.
	if w := postWithPhone(r, "/api/inbox/read", "0622000099", map[string]interface{}{"ids": []string{}}); w.Code != http.StatusNoContent {
		t.Fatalf("empty mark read expected 204, got %d", w.Code)
	}
	if got := getInbox(t, r, "0622000099"); got.Unread != 1 {
		t.Errorf("empty ids must not mark anything read; unread=%d", got.Unread)
	}

	// The owner marks it read by id.
	if w := postWithPhone(r, "/api/inbox/read", "0622000099", map[string]interface{}{"ids": []string{it.ID}}); w.Code != http.StatusNoContent {
		t.Fatalf("mark read expected 204, got %d", w.Code)
	}
	got := getInbox(t, r, "0622000099")
	if got.Unread != 0 || !got.Items[0].Read {
		t.Errorf("expected item read, got unread=%d read=%v", got.Unread, got.Items[0].Read)
	}
}

func TestInbox_MarkAllRead(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	// Two separate requests → two offers → two inbox items for the searcher.
	for _, dest := range []string{"Crest", "Die"} {
		w := postJSON(r, "/api/requests", map[string]interface{}{
			"searcher_name": "Bob", "phone": "0622000099", "origin": "Saillans", "destination": dest,
		})
		var created map[string]interface{}
		_ = json.NewDecoder(w.Body).Decode(&created)
		postJSON(r, "/api/requests/"+created["ID"].(string)+"/offer-contact",
			map[string]interface{}{"phone": "0611000001", "name": "Alice"})
	}
	if got := getInbox(t, r, "0622000099"); got.Unread != 2 {
		t.Fatalf("expected 2 unread, got %d", got.Unread)
	}
	if w := postWithPhone(r, "/api/inbox/read", "0622000099", map[string]interface{}{"all": true}); w.Code != http.StatusNoContent {
		t.Fatalf("mark all read expected 204, got %d", w.Code)
	}
	if got := getInbox(t, r, "0622000099"); got.Unread != 0 {
		t.Errorf("expected 0 unread after mark-all, got %d", got.Unread)
	}
}

func TestInbox_RequiresPhone(t *testing.T) {
	r := setupRouter()
	if w := getReq(r, "/api/inbox"); w.Code != http.StatusUnauthorized {
		t.Errorf("GET /inbox without X-Phone: expected 401, got %d", w.Code)
	}
	if w := postJSON(r, "/api/inbox/read", map[string]interface{}{}); w.Code != http.StatusUnauthorized {
		t.Errorf("POST /inbox/read without X-Phone: expected 401, got %d", w.Code)
	}
}

// Deleting a request removes every inbox item recorded about it.
func TestInbox_DeletingRequestRemovesItsNotifications(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	w := postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Bob", "phone": "0622000099", "origin": "Saillans", "destination": "Crest",
	})
	var created map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&created)
	requestID := created["ID"].(string)
	postJSON(r, "/api/requests/"+requestID+"/offer-contact", map[string]interface{}{"phone": "0611000001", "name": "Alice"})
	if got := getInbox(t, r, "0622000099"); len(got.Items) != 1 {
		t.Fatalf("expected 1 item before delete, got %d", len(got.Items))
	}

	if w := deleteJSONBody(r, "/api/requests/"+requestID, map[string]interface{}{"phone": "0622000099"}); w.Code/100 != 2 {
		t.Fatalf("DELETE request expected 2xx, got %d: %s", w.Code, w.Body.String())
	}
	if got := getInbox(t, r, "0622000099"); len(got.Items) != 0 {
		t.Errorf("expected inbox items about the deleted request to be gone, got %d", len(got.Items))
	}
}

// The privacy case: when a driver deletes their ride, their name and route
// disappear from the searchers' inboxes too.
func TestInbox_DeletingRideRemovesItFromSearchersInbox(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	// Searcher's alert, then a matching ride → ride_match in the searcher's inbox.
	if w := postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Bob", "phone": "0622000099", "origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	}); w.Code != http.StatusCreated {
		t.Fatalf("POST /requests expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "0611000001", "origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /rides expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var ride map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&ride)
	rideID := ride["ID"].(string)

	before := getInbox(t, r, "0622000099")
	if len(before.Items) != 1 || before.Items[0].Kind != "ride_match" || before.Items[0].ContactName != "Alice" {
		t.Fatalf("expected Alice's ride_match in Bob's inbox, got %+v", before.Items)
	}

	if w := deleteJSONBody(r, "/api/rides/"+rideID, map[string]interface{}{"phone": "0611000001"}); w.Code/100 != 2 {
		t.Fatalf("DELETE ride expected 2xx, got %d: %s", w.Code, w.Body.String())
	}
	if after := getInbox(t, r, "0622000099"); len(after.Items) != 0 {
		t.Errorf("deleted ride must leave no trace in Bob's inbox, got %+v", after.Items)
	}
}

func deleteJSONBody(r *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}
