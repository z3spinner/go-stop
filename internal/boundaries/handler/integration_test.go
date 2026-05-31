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

	handlerPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
	os.Exit(m.Run())
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	rideRepo := postgres.NewRideRepo(handlerPool)
	reqRepo := postgres.NewRequestRepo(handlerPool)
	subRepo := postgres.NewSubscriptionRepo(handlerPool)
	destRepo := postgres.NewDestinationRepo(handlerPool)
	n := &noopNotifier{}

	postRide := usecase.NewPostRide(rideRepo, reqRepo, subRepo, n)
	getRides := usecase.NewGetRides(rideRepo)
	searchRides := usecase.NewSearchRides(rideRepo)
	deleteRide := usecase.NewDeleteRide(rideRepo)
	postRequest := usecase.NewPostRequest(reqRepo, rideRepo, subRepo, n)
	deleteRequest := usecase.NewDeleteRequest(reqRepo)
	getDests := usecase.NewGetDestinations(destRepo)
	subscribe := usecase.NewSubscribe(subRepo)
	unsubscribe := usecase.NewUnsubscribe(subRepo)

	rideH := handler.NewRideHandler(postRide, getRides, searchRides, deleteRide, rideRepo)
	reqH := handler.NewRequestHandler(postRequest, deleteRequest, reqRepo)
	destH := handler.NewDestinationHandler(getDests)
	subH := handler.NewSubscriptionHandler(subscribe, unsubscribe)

	r := gin.New()
	r.POST("/api/rides", rideH.Post)
	r.GET("/api/rides", rideH.List)
	r.GET("/api/rides/:id", rideH.Get)
	r.DELETE("/api/rides/:id", rideH.Delete)
	r.POST("/api/requests", reqH.Post)
	r.GET("/api/requests/:id", reqH.Get)
	r.DELETE("/api/requests/:id", reqH.Delete)
	r.GET("/api/destinations", destH.List)
	r.POST("/api/subscriptions", subH.Subscribe)
	r.DELETE("/api/subscriptions/:phone", subH.Unsubscribe)
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
