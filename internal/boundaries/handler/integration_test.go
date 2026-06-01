//go:build integration

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/boundaries/handler"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/usecase"
)

var handlerPool *pgxpool.Pool

type noopNotifier struct{}

func (n *noopNotifier) Send(_ domain.Subscription, _ domain.Message) error { return nil }

var _ interface{ Send(domain.Subscription, domain.Message) error } = (*noopNotifier)(nil)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		os.Exit(0)
	}

	var err error
	handlerPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic("connect test db: " + err.Error())
	}
	defer handlerPool.Close()

	var truncErr error
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		_, truncErr = handlerPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions, ride_stats, interests`)
		if truncErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if truncErr != nil {
		panic("schema not ready after 30s: " + truncErr.Error())
	}

	os.Exit(m.Run())
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	rideRepo := postgres.NewRideRepo(handlerPool, 60)
	reqRepo := postgres.NewRequestRepo(handlerPool)
	subRepo := postgres.NewSubscriptionRepo(handlerPool)
	destRepo := postgres.NewDestinationRepo(handlerPool)
	n := &noopNotifier{}

	postRide := usecase.NewPostRide(rideRepo, reqRepo, subRepo, n)
	getRides := usecase.NewGetRides(rideRepo)
	getMyRides := usecase.NewGetMyRides(rideRepo)
	searchRides := usecase.NewSearchRides(rideRepo)
	deleteRide := usecase.NewDeleteRide(rideRepo)
	postRequest := usecase.NewPostRequest(reqRepo, rideRepo, subRepo, n)
	getMyRequests := usecase.NewGetMyRequests(reqRepo)
	deleteRequest := usecase.NewDeleteRequest(reqRepo)
	getDests := usecase.NewGetDestinations(destRepo)
	subscribe := usecase.NewSubscribe(subRepo)
	unsubscribe := usecase.NewUnsubscribe(subRepo)
	statRepo := postgres.NewStatRepo(handlerPool)
	interestRepo := postgres.NewInterestRepo(handlerPool)
	getMatchingRequests := usecase.NewGetMatchingRequests(rideRepo, reqRepo)
	recordFeedback := usecase.NewRecordFeedback(rideRepo, statRepo)
	getStats := usecase.NewGetStats(statRepo)
	expressInterest := usecase.NewExpressInterest(rideRepo, interestRepo, subRepo, n)
	acceptInterest := usecase.NewAcceptInterest(interestRepo, rideRepo, subRepo, n)
	getInterestContact := usecase.NewGetInterestContact(interestRepo, rideRepo)
	feedbackH := handler.NewFeedbackHandler(recordFeedback)
	statsH := handler.NewStatsHandler(getStats)
	interestH := handler.NewInterestHandler(expressInterest, acceptInterest, getInterestContact)

	rideH := handler.NewRideHandler(postRide, getRides, getMyRides, searchRides, deleteRide, getMatchingRequests, statRepo, interestRepo, rideRepo)
	reqH := handler.NewRequestHandler(postRequest, getMyRequests, deleteRequest, reqRepo)
	destH := handler.NewDestinationHandler(getDests)
	subH := handler.NewSubscriptionHandler(subscribe, unsubscribe)

	r := gin.New()
	r.POST("/api/rides", rideH.Post)
	r.GET("/api/rides", rideH.List)
	r.GET("/api/rides/:id", rideH.Get)
	r.DELETE("/api/rides/:id", rideH.Delete)
	r.GET("/api/rides/:id/requests", rideH.ListMatchingRequests)
	r.POST("/api/rides/:id/feedback", feedbackH.Post)
	r.GET("/api/rides/:id/interests", rideH.ListInterests)
	r.POST("/api/rides/:id/interest", interestH.Express)
	r.POST("/api/interests/:id/accept", interestH.Accept)
	r.GET("/api/interests/:id/contact", interestH.GetContact)
	r.POST("/api/requests", reqH.Post)
	r.GET("/api/requests", reqH.List)
	r.GET("/api/requests/:id", reqH.Get)
	r.DELETE("/api/requests/:id", reqH.Delete)
	r.GET("/api/destinations", destH.List)
	r.POST("/api/subscriptions", subH.Subscribe)
	r.DELETE("/api/subscriptions/:phone", subH.Unsubscribe)
	r.GET("/api/stats", statsH.Get)
	return r
}

func truncateAll(t *testing.T) {
	t.Helper()
	if _, err := handlerPool.Exec(context.Background(),
		`TRUNCATE rides, requests, subscriptions, ride_stats, interests`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func postJSON(r *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestHTTP_PostAndGetRides(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-0001",
		"origin": "Village A", "destination": "Station",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the response body contains the assigned ID
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	if created["ID"] == "" || created["ID"] == nil {
		t.Error("expected non-empty ID in 201 response body")
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/rides", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var rides []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &rides)
	if len(rides) != 1 {
		t.Errorf("expected 1 ride, got %d", len(rides))
	}
}

func TestHTTP_DeleteRide_WrongPhone_Returns403(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-0001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["ID"].(string)

	b, _ := json.Marshal(map[string]string{"phone": "555-9999"})
	w2 := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/rides/"+id, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}

func TestHTTP_Destinations_ReturnsSortedUnique(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "A", "phone": "1",
		"origin": "Village A", "destination": "Station",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "B", "phone": "2",
		"origin": "Town B", "destination": "Station",
		"departure_at": "2030-06-01T10:00:00Z", "flexibility": 0,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/destinations", nil)
	r.ServeHTTP(w, req)

	var dests []string
	json.Unmarshal(w.Body.Bytes(), &dests)
	if len(dests) != 3 {
		t.Errorf("expected 3 destinations, got %d: %v", len(dests), dests)
	}
}

func TestHTTP_MyRides_XPhoneHeader_FiltersToOwnerOnly(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	// Alice posts a ride
	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "5551001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	// Bob posts a ride
	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "5552001",
		"origin": "C", "destination": "D",
		"departure_at": "2030-06-01T10:00:00Z", "flexibility": 0,
	})

	// Fetch with Alice's phone — must only return Alice's ride
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/rides", nil)
	req.Header.Set("X-Phone", "5551001")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var rides []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &rides)
	if len(rides) != 1 {
		t.Errorf("expected 1 ride for Alice, got %d — X-Phone filter not working", len(rides))
	}
	if len(rides) > 0 && rides[0]["DriverName"] != "Alice" {
		t.Errorf("expected Alice's ride, got driver: %v", rides[0]["DriverName"])
	}
}

func TestHTTP_MyAlerts_XPhoneHeader_FiltersToOwnerOnly(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	// Carol posts a request
	postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Carol", "phone": "5553001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	// Dave posts a request
	postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Dave", "phone": "5554001",
		"origin": "C", "destination": "D",
		"departure_at": "2030-06-01T10:00:00Z", "flexibility": 0,
	})

	// Fetch with Carol's phone — must only return Carol's request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/requests", nil)
	req.Header.Set("X-Phone", "5553001")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var reqs []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &reqs)
	if len(reqs) != 1 {
		t.Errorf("expected 1 request for Carol, got %d — X-Phone filter not working", len(reqs))
	}
	if len(reqs) > 0 && reqs[0]["SearcherName"] != "Carol" {
		t.Errorf("expected Carol's request, got: %v", reqs[0]["SearcherName"])
	}
}

func TestHTTP_MyRides_NoXPhoneHeader_ReturnsAllRides(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "5551001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "5552001",
		"origin": "C", "destination": "D",
		"departure_at": "2030-06-01T10:00:00Z", "flexibility": 0,
	})

	// Without X-Phone header, all active rides are returned
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/rides", nil)
	r.ServeHTTP(w, req)

	var rides []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &rides)
	if len(rides) != 2 {
		t.Errorf("expected 2 rides without filter, got %d", len(rides))
	}
}

func TestHTTP_Feedback_RecordsStatAndMarksFeedbackGiven(t *testing.T) {
	truncateAll(t)
	// Also truncate ride_stats
	handlerPool.Exec(context.Background(), `TRUNCATE ride_stats`)
	r := setupRouter()

	// Post a ride
	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-0001",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["ID"].(string)

	// Submit positive feedback
	w2 := postJSON(r, "/api/rides/"+id+"/feedback", map[string]interface{}{
		"phone": "555-0001",
		"taken": true,
	})
	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w2.Code, w2.Body.String())
	}

	// Stats should now show the route
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/api/stats", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w3.Code)
	}
	var stats map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &stats)
	if stats["total_this_week"].(float64) != 1 {
		t.Errorf("expected total_this_week=1, got %v", stats["total_this_week"])
	}
}

func TestHTTP_Feedback_WrongPhone_Returns403(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-0001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["ID"].(string)

	w2 := postJSON(r, "/api/rides/"+id+"/feedback", map[string]interface{}{
		"phone": "555-9999",
		"taken": true,
	})
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}

func TestHTTP_Interest_ExpressCreatesRecord(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "5550001",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	})
	var ride map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ride)
	rideID := ride["ID"].(string)

	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550002",
	})
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
	var interest map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &interest)
	if interest["id"] == nil || interest["id"] == "" {
		t.Error("expected interest ID in response")
	}
	if interest["status"] != "pending" {
		t.Errorf("expected pending status, got %v", interest["status"])
	}
	if interest["searcher_phone"] != nil || interest["phone"] != nil {
		t.Error("phone must not appear in express-interest response")
	}
}

func TestHTTP_Interest_DriverCannotBeSearcher(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "5550001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	var ride map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ride)
	rideID := ride["ID"].(string)

	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550001",
	})
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}

func TestHTTP_Interest_AcceptRevealsPhonesCorrectly(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	// Driver posts ride
	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "5550001",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	})
	var ride map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ride)
	rideID := ride["ID"].(string)

	// Searcher expresses interest
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550002",
	})
	var interest map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &interest)
	interestID := interest["id"].(string)

	// Contact endpoint returns error while pending
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/api/interests/"+interestID+"/contact", nil)
	req3.Header.Set("X-Phone", "5550002")
	r.ServeHTTP(w3, req3)
	if w3.Code == http.StatusOK {
		t.Error("expected non-200 while interest is pending")
	}

	// Driver accepts
	w4 := postJSON(r, "/api/interests/"+interestID+"/accept", map[string]interface{}{
		"phone": "5550001",
	})
	if w4.Code != http.StatusOK {
		t.Fatalf("expected 200 on accept, got %d: %s", w4.Code, w4.Body.String())
	}
	var acceptResp map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &acceptResp)
	if acceptResp["searcher_phone"] != "5550001" && acceptResp["searcher_phone"] != "5550002" {
		t.Errorf("driver should receive searcher phone, got %v", acceptResp["searcher_phone"])
	}
	if acceptResp["searcher_phone"] != "5550002" {
		t.Errorf("driver should receive searcher phone 5550002, got %v", acceptResp["searcher_phone"])
	}

	// Searcher can now get driver's phone
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodGet, "/api/interests/"+interestID+"/contact", nil)
	req5.Header.Set("X-Phone", "5550002")
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Fatalf("expected 200 for searcher contact, got %d: %s", w5.Code, w5.Body.String())
	}
	var contactResp map[string]interface{}
	json.Unmarshal(w5.Body.Bytes(), &contactResp)
	if contactResp["phone"] != "5550001" {
		t.Errorf("searcher should receive driver phone 5550001, got %v", contactResp["phone"])
	}

	// Stranger gets 403
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodGet, "/api/interests/"+interestID+"/contact", nil)
	req6.Header.Set("X-Phone", "5550099")
	r.ServeHTTP(w6, req6)
	if w6.Code != http.StatusForbidden {
		t.Errorf("expected 403 for stranger, got %d", w6.Code)
	}
}

func TestHTTP_PublicRideList_StripsPIIFields(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "5550001",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})

	// Public request (no X-Phone)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/rides", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var rides []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &rides)
	if len(rides) != 1 {
		t.Fatalf("expected 1 ride, got %d", len(rides))
	}
	if rides[0]["Phone"] != nil {
		t.Errorf("Phone must not appear in public ride list, got %v", rides[0]["Phone"])
	}
	// DriverName is intentionally public (mutual interest feature)
	if rides[0]["DriverName"] == nil {
		t.Error("DriverName must be present in public ride list")
	}
	if rides[0]["Origin"] == nil {
		t.Error("Origin must be present in public ride list")
	}
}

func TestHTTP_Interest_WrongDriverPhoneReturns403(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "5550001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	var ride map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ride)
	rideID := ride["ID"].(string)

	// Searcher expresses interest
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550002",
	})
	var interest map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &interest)
	interestID := interest["id"].(string)

	// Wrong driver tries to accept
	w3 := postJSON(r, "/api/interests/"+interestID+"/accept", map[string]interface{}{
		"phone": "5550099",
	})
	if w3.Code != http.StatusForbidden {
		t.Errorf("expected 403 for wrong driver, got %d", w3.Code)
	}
}

func TestHTTP_Alert_TimeMode_MatchesOverlappingRide(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	// Post a time-mode alert for 09:00 ±30 min on 2030-06-01
	w := postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Alice", "phone": "5560001",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Ride at 09:15 — within the ±30 min window
	w2 := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "5560002",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:15:00Z", "flexibility": 0,
	})
	var ride map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &ride)

	req2, _ := http.NewRequest(http.MethodGet, "/api/rides/"+ride["ID"].(string)+"/requests", nil)
	req2.Header.Set("X-Phone", "5560002")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req2)
	var matching []map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &matching)
	if len(matching) != 1 {
		t.Errorf("time mode: expected 1 match, got %d", len(matching))
	}
}

func TestHTTP_Alert_DayMode_MatchesAnyTimeOnDate(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	// Post a day-mode alert for 2030-06-01 (any time)
	w := postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Alice", "phone": "5561001",
		"origin": "Saillans", "destination": "Crest",
		"departure_date": "2030-06-01",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Ride early morning — should match (any time on that day)
	w2 := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "5561002",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T06:00:00Z", "flexibility": 0,
	})
	var ride map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &ride)

	req2, _ := http.NewRequest(http.MethodGet, "/api/rides/"+ride["ID"].(string)+"/requests", nil)
	req2.Header.Set("X-Phone", "5561002")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req2)
	var matching []map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &matching)
	if len(matching) != 1 {
		t.Errorf("day mode: expected 1 match for any time on date, got %d", len(matching))
	}

	// Ride on a different day — must NOT match
	w4 := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "5561002",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-02T06:00:00Z", "flexibility": 0,
	})
	var ride2 map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &ride2)

	req3, _ := http.NewRequest(http.MethodGet, "/api/rides/"+ride2["ID"].(string)+"/requests", nil)
	req3.Header.Set("X-Phone", "5561002")
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req3)
	var noMatch []map[string]interface{}
	json.Unmarshal(w5.Body.Bytes(), &noMatch)
	if len(noMatch) != 0 {
		t.Errorf("day mode: expected 0 matches for different date, got %d", len(noMatch))
	}
}

func TestHTTP_Alert_AnytimeMode_MatchesAnyRideOnRoute(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	// Post an anytime alert — no date, no time
	w := postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Alice", "phone": "5562001",
		"origin": "Saillans", "destination": "Crest",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Any ride on this route should match
	w2 := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "5562002",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-07-15T14:00:00Z", "flexibility": 0,
	})
	var ride map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &ride)

	req2, _ := http.NewRequest(http.MethodGet, "/api/rides/"+ride["ID"].(string)+"/requests", nil)
	req2.Header.Set("X-Phone", "5562002")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req2)
	var matching []map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &matching)
	if len(matching) != 1 {
		t.Errorf("anytime mode: expected 1 match, got %d", len(matching))
	}

	// Different route — must NOT match
	w4 := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "5562002",
		"origin": "Saillans", "destination": "Die",
		"departure_at": "2030-07-15T14:00:00Z", "flexibility": 0,
	})
	var ride2 map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &ride2)

	req3, _ := http.NewRequest(http.MethodGet, "/api/rides/"+ride2["ID"].(string)+"/requests", nil)
	req3.Header.Set("X-Phone", "5562002")
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req3)
	var noMatch []map[string]interface{}
	json.Unmarshal(w5.Body.Bytes(), &noMatch)
	if len(noMatch) != 0 {
		t.Errorf("anytime mode: expected 0 matches for different route, got %d", len(noMatch))
	}
}
