// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ── helper: DELETE with JSON body ─────────────────────────────────────────────
func deleteJSON(r *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// ── contact offer integration tests ─────────────────────────────────────────

func TestOfferContact_CreatesAndSearcherCanList(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	// Post a search request.
	reqBody := map[string]interface{}{
		"searcher_name": "Bob",
		"phone":         "0622000099",
		"origin":        "Saillans",
		"destination":   "Crest",
	}
	w := postJSON(r, "/api/requests", reqBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /requests expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	requestID := created["ID"].(string)

	// A third party offers their contact.
	offerBody := map[string]interface{}{
		"phone": "0611000001",
		"name":  "Alice",
	}
	w2 := postJSON(r, "/api/requests/"+requestID+"/offer-contact", offerBody)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("POST offer-contact expected 204, got %d: %s", w2.Code, w2.Body.String())
	}

	// Searcher lists offers on their request.
	w3 := getWithPhone(r, "/api/requests/"+requestID+"/offers", "0622000099")
	if w3.Code != http.StatusOK {
		t.Fatalf("GET /offers expected 200, got %d: %s", w3.Code, w3.Body.String())
	}
	var offers []map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&offers)
	if len(offers) != 1 {
		t.Fatalf("expected 1 offer, got %d", len(offers))
	}
	if offers[0]["offerer_name"] != "Alice" {
		t.Errorf("expected offerer_name Alice, got %v", offers[0]["offerer_name"])
	}
	if offers[0]["offerer_phone"] != "0611000001" {
		t.Errorf("expected offerer_phone 0611000001, got %v", offers[0]["offerer_phone"])
	}
}

func TestOfferContact_NonOwnerCannotListOffers(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	reqBody := map[string]interface{}{
		"searcher_name": "Bob",
		"phone":         "0622000099",
		"origin":        "Saillans",
		"destination":   "Crest",
	}
	w := postJSON(r, "/api/requests", reqBody)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	requestID := created["ID"].(string)

	// Offer from a third party.
	postJSON(r, "/api/requests/"+requestID+"/offer-contact", map[string]interface{}{
		"phone": "0611000001",
		"name":  "Alice",
	})

	// A different phone tries to list offers — should be 403.
	w2 := getWithPhone(r, "/api/requests/"+requestID+"/offers", "0699999999")
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-owner, got %d", w2.Code)
	}
}

func TestOfferContact_RequiresPhone(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	reqBody := map[string]interface{}{
		"searcher_name": "Bob",
		"phone":         "0622000099",
		"origin":        "Saillans",
		"destination":   "Crest",
	}
	w := postJSON(r, "/api/requests", reqBody)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	requestID := created["ID"].(string)

	// No phone field → 400.
	w2 := postJSON(r, "/api/requests/"+requestID+"/offer-contact", map[string]interface{}{
		"name": "Alice",
	})
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when phone missing, got %d", w2.Code)
	}
}

func TestOfferContact_RequiresName(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	reqBody := map[string]interface{}{
		"searcher_name": "Bob",
		"phone":         "0622000099",
		"origin":        "Saillans",
		"destination":   "Crest",
	}
	w := postJSON(r, "/api/requests", reqBody)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	requestID := created["ID"].(string)

	// Empty name → 400.
	w2 := postJSON(r, "/api/requests/"+requestID+"/offer-contact", map[string]interface{}{
		"phone": "0611000001",
		"name":  "",
	})
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when name empty, got %d", w2.Code)
	}
}

func TestOfferContact_SearcherCannotOfferToOwnRequest(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	reqBody := map[string]interface{}{
		"searcher_name": "Bob",
		"phone":         "0622000099",
		"origin":        "Saillans",
		"destination":   "Crest",
	}
	w := postJSON(r, "/api/requests", reqBody)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	requestID := created["ID"].(string)

	// Same phone as the searcher → 403.
	w2 := postJSON(r, "/api/requests/"+requestID+"/offer-contact", map[string]interface{}{
		"phone": "0622000099",
		"name":  "Bob",
	})
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 when searcher offers to own request, got %d", w2.Code)
	}
}

func TestOfferContact_DuplicateOfferIsIdempotent(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	reqBody := map[string]interface{}{
		"searcher_name": "Bob",
		"phone":         "0622000099",
		"origin":        "Saillans",
		"destination":   "Crest",
	}
	w := postJSON(r, "/api/requests", reqBody)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	requestID := created["ID"].(string)

	offerBody := map[string]interface{}{"phone": "0611000001", "name": "Alice"}

	// First offer.
	w1 := postJSON(r, "/api/requests/"+requestID+"/offer-contact", offerBody)
	if w1.Code != http.StatusNoContent {
		t.Fatalf("first offer expected 204, got %d", w1.Code)
	}
	// Duplicate offer — should also succeed.
	w2 := postJSON(r, "/api/requests/"+requestID+"/offer-contact", offerBody)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("duplicate offer expected 204, got %d", w2.Code)
	}

	// Only one row in the list.
	w3 := getWithPhone(r, "/api/requests/"+requestID+"/offers", "0622000099")
	var offers []map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&offers)
	if len(offers) != 1 {
		t.Errorf("expected exactly 1 offer after duplicate, got %d", len(offers))
	}
}

func TestListContactOffers_RequiresXPhone(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	// GET /offers without X-Phone header → 401.
	w := getReq(r, "/api/requests/some-id/offers")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when X-Phone missing, got %d", w.Code)
	}
}

func TestOfferContact_UnknownRequestReturns404(t *testing.T) {
	r := setupRouter()
	truncateAll(t)

	w := postJSON(r, "/api/requests/00000000-0000-0000-0000-000000000000/offer-contact", map[string]interface{}{
		"phone": "0611000001",
		"name":  "Alice",
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown request, got %d", w.Code)
	}
}
