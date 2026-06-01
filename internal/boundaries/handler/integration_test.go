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
		_, truncErr = handlerPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
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
	getMatchingRequests := usecase.NewGetMatchingRequests(rideRepo, reqRepo)
	recordFeedback := usecase.NewRecordFeedback(rideRepo, statRepo)
	getStats := usecase.NewGetStats(statRepo)
	feedbackH := handler.NewFeedbackHandler(recordFeedback)
	statsH := handler.NewStatsHandler(getStats)

	rideH := handler.NewRideHandler(postRide, getRides, getMyRides, searchRides, deleteRide, getMatchingRequests, rideRepo)
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
	handlerPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
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
		"driver_name": "Alice", "phone": "555-alice",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	// Bob posts a ride
	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "555-bob",
		"origin": "C", "destination": "D",
		"departure_at": "2030-06-01T10:00:00Z", "flexibility": 0,
	})

	// Fetch with Alice's phone — must only return Alice's ride
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/rides", nil)
	req.Header.Set("X-Phone", "555-alice")
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
		"searcher_name": "Carol", "phone": "555-carol",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	// Dave posts a request
	postJSON(r, "/api/requests", map[string]interface{}{
		"searcher_name": "Dave", "phone": "555-dave",
		"origin": "C", "destination": "D",
		"departure_at": "2030-06-01T10:00:00Z", "flexibility": 0,
	})

	// Fetch with Carol's phone — must only return Carol's request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/requests", nil)
	req.Header.Set("X-Phone", "555-carol")
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
		"driver_name": "Alice", "phone": "555-alice",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Bob", "phone": "555-bob",
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
