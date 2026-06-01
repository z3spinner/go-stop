package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// publicRide is returned for public search/feed requests. Phone is absent; DriverName is visible.
type publicRide struct {
	ID            string    `json:"ID"`
	DriverName    string    `json:"DriverName"`
	Origin        string    `json:"Origin"`
	Destination   string    `json:"Destination"`
	Date          time.Time `json:"Date"`
	DepartureAt   time.Time `json:"DepartureAt"`
	Flexibility   int       `json:"Flexibility"`
	PostedAt      time.Time `json:"PostedAt"`
	ExpiresAt     time.Time `json:"ExpiresAt"`
	FeedbackGiven bool      `json:"FeedbackGiven"`
}

func toPublicRides(rides []domain.Ride) []publicRide {
	out := make([]publicRide, len(rides))
	for i, r := range rides {
		out[i] = publicRide{
			ID:            r.ID,
			DriverName:    r.DriverName,
			Origin:        r.Origin,
			Destination:   r.Destination,
			Date:          r.Date,
			DepartureAt:   r.DepartureAt,
			Flexibility:   int(r.Flexibility),
			PostedAt:      r.PostedAt,
			ExpiresAt:     r.ExpiresAt,
			FeedbackGiven: r.FeedbackGiven,
		}
	}
	return out
}

type RideHandler struct {
	postRide             *usecase.PostRide
	getRides             *usecase.GetRides
	getMyRides           *usecase.GetMyRides
	searchRides          *usecase.SearchRides
	deleteRide           *usecase.DeleteRide
	getMatchingRequests  *usecase.GetMatchingRequests
	statRepo             repository.StatRepository
	interestRepo         repository.InterestRepository
	rideRepo             repository.RideRepository
}

func NewRideHandler(
	postRide *usecase.PostRide,
	getRides *usecase.GetRides,
	getMyRides *usecase.GetMyRides,
	searchRides *usecase.SearchRides,
	deleteRide *usecase.DeleteRide,
	getMatchingRequests *usecase.GetMatchingRequests,
	statRepo repository.StatRepository,
	interestRepo repository.InterestRepository,
	rideRepo repository.RideRepository,
) *RideHandler {
	return &RideHandler{
		postRide:            postRide,
		getRides:            getRides,
		getMyRides:          getMyRides,
		searchRides:         searchRides,
		deleteRide:          deleteRide,
		getMatchingRequests: getMatchingRequests,
		statRepo:            statRepo,
		interestRepo:        interestRepo,
		rideRepo:            rideRepo,
	}
}

type postRideRequest struct {
	DriverName  string `json:"driver_name" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	Origin      string `json:"origin" binding:"required"`
	Destination string `json:"destination" binding:"required"`
	DepartureAt string `json:"departure_at" binding:"required"`
	Flexibility int    `json:"flexibility"`
}

func (h *RideHandler) Post(c *gin.Context) {
	var req postRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dept, err := time.Parse(time.RFC3339, req.DepartureAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid departure_at, use RFC3339"})
		return
	}
	ride := domain.Ride{
		DriverName:  req.DriverName,
		Phone:       normalizePhone(req.Phone),
		Origin:      normalizeLocation(req.Origin),
		Destination: normalizeLocation(req.Destination),
		DepartureAt: dept,
		Flexibility: domain.Flexibility(req.Flexibility),
	}
	saved, err := h.postRide.Execute(ride)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Record ride-posted event asynchronously (best-effort)
	if h.statRepo != nil {
		go func() { _ = h.statRepo.RecordRide(ride.Origin, ride.Destination) }()
	}
	c.JSON(http.StatusCreated, saved)
}

func (h *RideHandler) List(c *gin.Context) {
	origin := normalizeLocation(c.Query("origin"))
	destination := normalizeLocation(c.Query("destination"))
	phone := normalizePhone(c.GetHeader("X-Phone"))

	var rides []domain.Ride
	var err error
	switch {
	case phone != "":
		rides, err = h.getMyRides.Execute(phone)
	case origin != "" && destination != "":
		rides, err = h.searchRides.Execute(origin, destination)
		// Record search event asynchronously (best-effort, never blocks the response)
		if h.statRepo != nil {
			go func() { _ = h.statRepo.RecordSearch(origin, destination) }()
		}
	default:
		rides, err = h.getRides.Execute()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if phone != "" {
		c.JSON(http.StatusOK, rides)
	} else {
		c.JSON(http.StatusOK, toPublicRides(rides))
	}
}

func (h *RideHandler) Get(c *gin.Context) {
	ride, err := h.rideRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, ride)
}

func (h *RideHandler) ListMatchingRequests(c *gin.Context) {
	// Require the driver's phone via X-Phone header — same lightweight auth as delete.
	phone := c.GetHeader("X-Phone")
	phone = normalizePhone(phone)
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	ride, err := h.rideRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if ride.Phone != phone {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}
	requests, err := h.getMatchingRequests.Execute(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, requests)
}

type deleteRideRequest struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *RideHandler) Delete(c *gin.Context) {
	var req deleteRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.deleteRide.Execute(c.Param("id"), normalizePhone(req.Phone)); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *RideHandler) ListInterests(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	ride, err := h.rideRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if ride.Phone != phone {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}
	interests, err := h.interestRepo.FindByRide(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type interestResponse struct {
		SearcherName  string `json:"searcher_name,omitempty"`
		ID            string `json:"id"`
		Status        string `json:"status"`
		SearcherPhone string `json:"searcher_phone,omitempty"`
	}
	out := make([]interestResponse, len(interests))
	for i, interest := range interests {
		// Name is always shown to the driver; phone only after accepted
		out[i] = interestResponse{
			ID:           interest.ID,
			Status:       interest.Status,
			SearcherName: interest.SearcherName,
		}
		if interest.Status == "accepted" {
			out[i].SearcherPhone = interest.SearcherPhone
		}
	}
	c.JSON(http.StatusOK, out)
}
