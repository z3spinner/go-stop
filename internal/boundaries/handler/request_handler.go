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

type RequestHandler struct {
	postRequest    *usecase.PostRequest
	getMyRequests  *usecase.GetMyRequests
	deleteRequest  *usecase.DeleteRequest
	requestRepo    repository.RequestRepository
}

func NewRequestHandler(
	postRequest *usecase.PostRequest,
	getMyRequests *usecase.GetMyRequests,
	deleteRequest *usecase.DeleteRequest,
	requestRepo repository.RequestRepository,
) *RequestHandler {
	return &RequestHandler{
		postRequest:   postRequest,
		getMyRequests: getMyRequests,
		deleteRequest: deleteRequest,
		requestRepo:   requestRepo,
	}
}

func (h *RequestHandler) List(c *gin.Context) {
	phone := c.GetHeader("X-Phone")
	phone = normalizePhone(phone)
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone query parameter required"})
		return
	}
	requests, err := h.getMyRequests.Execute(phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, requests)
}

type postRequestBody struct {
	SearcherName    string `json:"searcher_name" binding:"required"`
	Phone           string `json:"phone" binding:"required"`
	Origin          string `json:"origin" binding:"required"`
	Destination     string `json:"destination" binding:"required"`
	DepartureAt     string `json:"departure_at"`   // RFC3339 — specific time mode
	DepartureDate   string `json:"departure_date"` // YYYY-MM-DD — day mode
	DepartureTime   string `json:"departure_time"` // HH:MM — daily mode (any day at this time)
	Flexibility     int    `json:"flexibility"`
}

func (h *RequestHandler) Post(c *gin.Context) {
	var body postRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := domain.Request{
		SearcherName: body.SearcherName,
		Phone:        normalizePhone(body.Phone),
		Origin:       normalizeLocation(body.Origin),
		Destination:  normalizeLocation(body.Destination),
		Flexibility:  domain.Flexibility(body.Flexibility),
	}
	switch {
	case body.DepartureAt != "":
		dept, err := time.Parse(time.RFC3339, body.DepartureAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid departure_at, use RFC3339"})
			return
		}
		req.DepartureAt = dept
	case body.DepartureDate != "":
		d, err := time.Parse("2006-01-02", body.DepartureDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid departure_date, use YYYY-MM-DD"})
			return
		}
		req.Date = d // day mode: Date set, DepartureAt stays zero
	case body.DepartureTime != "":
		tt, err := time.Parse("15:04", body.DepartureTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid departure_time, use HH:MM"})
			return
		}
		// Daily mode: Date stays zero, DepartureAt carries only the time (sentinel date 1970-01-01).
		req.DepartureAt = time.Date(1970, 1, 1, tt.Hour(), tt.Minute(), 0, 0, time.UTC)
	// neither → anytime: both Date and DepartureAt remain zero
	}
	saved, err := h.postRequest.Execute(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, saved)
}

func (h *RequestHandler) Get(c *gin.Context) {
	req, err := h.requestRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	// Only the request owner may retrieve the full record (including their phone).
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" || phone != req.Phone {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, req)
}

type deleteRequestBody struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *RequestHandler) Delete(c *gin.Context) {
	var body deleteRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.deleteRequest.Execute(c.Param("id"), normalizePhone(body.Phone)); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
