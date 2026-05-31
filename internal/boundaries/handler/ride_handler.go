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

type RideHandler struct {
	postRide    *usecase.PostRide
	getRides    *usecase.GetRides
	getMyRides  *usecase.GetMyRides
	searchRides *usecase.SearchRides
	deleteRide  *usecase.DeleteRide
	rideRepo    repository.RideRepository
}

func NewRideHandler(
	postRide *usecase.PostRide,
	getRides *usecase.GetRides,
	getMyRides *usecase.GetMyRides,
	searchRides *usecase.SearchRides,
	deleteRide *usecase.DeleteRide,
	rideRepo repository.RideRepository,
) *RideHandler {
	return &RideHandler{
		postRide:    postRide,
		getRides:    getRides,
		getMyRides:  getMyRides,
		searchRides: searchRides,
		deleteRide:  deleteRide,
		rideRepo:    rideRepo,
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
		Phone:       req.Phone,
		Origin:      req.Origin,
		Destination: req.Destination,
		DepartureAt: dept,
		Flexibility: domain.Flexibility(req.Flexibility),
	}
	saved, err := h.postRide.Execute(ride)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, saved)
}

func (h *RideHandler) List(c *gin.Context) {
	origin := c.Query("origin")
	destination := c.Query("destination")
	phone := c.GetHeader("X-Phone")

	var rides []domain.Ride
	var err error
	switch {
	case phone != "":
		rides, err = h.getMyRides.Execute(phone)
	case origin != "" && destination != "":
		rides, err = h.searchRides.Execute(origin, destination)
	default:
		rides, err = h.getRides.Execute()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rides)
}

func (h *RideHandler) Get(c *gin.Context) {
	ride, err := h.rideRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, ride)
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
	if err := h.deleteRide.Execute(c.Param("id"), req.Phone); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
