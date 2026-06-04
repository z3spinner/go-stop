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

func toPublicRides(rides []domain.Ride) []PublicRide {
	out := make([]PublicRide, len(rides))
	for i, r := range rides {
		out[i] = PublicRide{
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

func attachInterestCounts(rides []PublicRide, interestRepo repository.InterestRepository) []PublicRide {
	if len(rides) == 0 {
		return rides
	}
	ids := make([]string, len(rides))
	for i, r := range rides {
		ids[i] = r.ID
	}
	counts, err := interestRepo.CountByRides(ids)
	if err != nil {
		return rides // best-effort: return without counts on error
	}
	for i := range rides {
		rides[i].InterestCount = counts[rides[i].ID]
	}
	return rides
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
	serviceTZ            *time.Location
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
	serviceTZ *time.Location,
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
		serviceTZ:           serviceTZ,
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

// Post creates a new ride offer.
// @ID       createRide
// @Tags     rides
// @Accept   json
// @Produce  json
// @Param    body  body  handler.PostRideBody  true  "Ride to create"
// @Success  201  {object}  domain.Ride
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /rides [post]
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

// List returns rides. With an X-Phone header it returns the caller's own rides
// (full domain.Ride incl. phone); otherwise the public feed/search of PublicRide.
// @ID       listRides
// @Tags     rides
// @Produce  json
// @Param    origin       query   string  false  "Origin filter"
// @Param    destination  query   string  false  "Destination filter"
// @Param    departure_at query   string  false  "Departure time (RFC3339)"
// @Param    search_date  query   string  false  "Search date (YYYY-MM-DD)"
// @Param    search_time  query   string  false  "Search time (HH:MM, local)"
// @Param    X-Phone      header  string  false  "Caller phone for my-rides mode"
// @Success  200  {array}  handler.PublicRide
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /rides [get]
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
		var searchDate, deptAt, searchTime time.Time
		if raw := c.Query("departure_at"); raw != "" {
			if parsed, err2 := time.Parse(time.RFC3339, raw); err2 == nil {
				deptAt = parsed
			}
		} else if raw := c.Query("search_date"); raw != "" {
			if parsed, err2 := time.Parse("2006-01-02", raw); err2 == nil {
				searchDate = parsed
			}
		} else if raw := c.Query("search_time"); raw != "" {
			// HH:MM local time — convert to UTC using the service timezone
			if parsed, err2 := time.Parse("15:04", raw); err2 == nil {
				loc := h.serviceTZ
				if loc == nil {
					loc = time.UTC
				}
				now := time.Now().In(loc)
				searchTime = time.Date(now.Year(), now.Month(), now.Day(),
					parsed.Hour(), parsed.Minute(), 0, 0, loc)
			}
		}
		rides, err = h.searchRides.Execute(origin, destination, searchDate, deptAt, searchTime)
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
		c.JSON(http.StatusOK, attachInterestCounts(toPublicRides(rides), h.interestRepo))
	}
}

// Get returns a single ride by ID (public — no phone).
// The detail page is publicly shareable, so the driver's phone is withheld here;
// it is revealed only through the accepted-interest contact flow.
// @ID       getRide
// @Tags     rides
// @Produce  json
// @Param    id  path  string  true  "Ride ID"
// @Success  200  {object}  handler.PublicRide
// @Failure  404  {object}  handler.ErrorResponse
// @Router   /rides/{id} [get]
func (h *RideHandler) Get(c *gin.Context) {
	ride, err := h.rideRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	pub := attachInterestCounts(toPublicRides([]domain.Ride{ride}), h.interestRepo)
	c.JSON(http.StatusOK, pub[0])
}

// ListMatchingRequests returns searcher alerts matching the driver's ride
// (phone-stripped). Requires the ride owner's X-Phone header.
// @ID       listRideRequests
// @Tags     rides
// @Produce  json
// @Param    id       path    string  true   "Ride ID"
// @Param    X-Phone  header  string  true   "Ride owner phone"
// @Success  200  {array}  handler.PublicRequest
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /rides/{id}/requests [get]
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
	// Strip phone — searchers who set up alerts have not consented to sharing
	// their number with every driver. Only the searcher's name and timing are shown.
	c.JSON(http.StatusOK, toPublicRequests(requests))
}

type deleteRideRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// Delete removes a ride owned by the caller.
// @ID       deleteRide
// @Tags     rides
// @Accept   json
// @Param    id    path  string                  true  "Ride ID"
// @Param    body  body  handler.DeleteRideBody  true  "Owner phone"
// @Success  204
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /rides/{id} [delete]
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

// ListInterests returns the interests expressed in the caller's ride (driver view).
// Requires the ride owner's X-Phone header.
// @ID       listRideInterests
// @Tags     interests
// @Produce  json
// @Param    id       path    string  true  "Ride ID"
// @Param    X-Phone  header  string  true  "Ride owner phone"
// @Success  200  {array}  handler.InterestListItem
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /rides/{id}/interests [get]
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
	out := make([]InterestListItem, len(interests))
	for i, interest := range interests {
		// Name always shown; phone shown for mutual-accepted only.
		// driver_shared = driver pinged searcher (one-way) — no phone shown to driver.
		out[i] = InterestListItem{
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
