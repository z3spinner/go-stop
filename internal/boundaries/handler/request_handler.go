// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

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
	postRequest              *usecase.PostRequest
	getMyRequests            *usecase.GetMyRequests
	getActiveRequests        *usecase.GetActiveRequests
	deleteRequest            *usecase.DeleteRequest
	pingSearcher             *usecase.PingSearcher
	offerContact             *usecase.OfferContact
	getRequestContactOffers  *usecase.GetRequestContactOffers
	requestRepo              repository.RequestRepository
	statRepo                 repository.StatRepository
}

func NewRequestHandler(
	postRequest *usecase.PostRequest,
	getMyRequests *usecase.GetMyRequests,
	getActiveRequests *usecase.GetActiveRequests,
	deleteRequest *usecase.DeleteRequest,
	pingSearcher *usecase.PingSearcher,
	offerContact *usecase.OfferContact,
	getRequestContactOffers *usecase.GetRequestContactOffers,
	requestRepo repository.RequestRepository,
	statRepo repository.StatRepository,
) *RequestHandler {
	return &RequestHandler{
		postRequest:             postRequest,
		getMyRequests:           getMyRequests,
		getActiveRequests:       getActiveRequests,
		deleteRequest:           deleteRequest,
		pingSearcher:            pingSearcher,
		offerContact:            offerContact,
		getRequestContactOffers: getRequestContactOffers,
		requestRepo:             requestRepo,
		statRepo:                statRepo,
	}
}

// toPublicRequests strips the searcher's phone for public / driver-facing lists.
func toPublicRequests(requests []domain.Request) []PublicRequest {
	out := make([]PublicRequest, len(requests))
	for i, r := range requests {
		out[i] = PublicRequest{
			ID:           r.ID,
			SearcherName: r.SearcherName,
			Origin:       r.Origin,
			Destination:  r.Destination,
			Date:         r.Date,
			DepartureAt:  r.DepartureAt,
			Flexibility:  int(r.Flexibility),
		}
	}
	return out
}

type pingSearcherBody struct {
	RideID string `json:"ride_id" binding:"required"`
}

// Ping notifies the searcher that a driver's ride matches their alert.
// POST /requests/:id/ping  (X-Phone = driver's phone)
// @ID       pingRequest
// @Tags     requests
// @Accept   json
// @Param    id       path    string                   true  "Request ID"
// @Param    X-Phone  header  string                   true  "Driver phone"
// @Param    body     body    handler.PingRequestBody  true  "Matching ride ID"
// @Success  204
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Router   /requests/{id}/ping [post]
func (h *RequestHandler) Ping(c *gin.Context) {
	driverPhone := normalizePhone(c.GetHeader("X-Phone"))
	if driverPhone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	var body pingSearcherBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	connected, err := h.pingSearcher.Execute(c.Param("id"), body.RideID, driverPhone)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	// Record the connection asynchronously (best-effort), only when the ping
	// actually created a new driver_shared interest so repeats don't double-count.
	if connected && h.statRepo != nil {
		go func() { _ = h.statRepo.RecordConnection() }()
	}
	c.Status(http.StatusNoContent)
}

// List returns the caller's own ride-search alerts.
// @ID       listRequests
// @Tags     requests
// @Produce  json
// @Param    X-Phone  header  string  false  "Searcher phone — when set, returns that searcher's own alerts (with phone); when absent, returns the public feed of active requests (phone stripped)"
// @Success  200  {array}  handler.PublicRequest
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /requests [get]
func (h *RequestHandler) List(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))

	// With a phone: the searcher's own alerts (full Request, includes phone).
	if phone != "" {
		requests, err := h.getMyRequests.Execute(phone)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, requests)
		return
	}

	// Without a phone: the public feed of all active requests (phone stripped),
	// mirroring GET /rides. Lets drivers browse demand on the home page.
	requests, err := h.getActiveRequests.Execute()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPublicRequests(requests))
}

type postRequestBody struct {
	SearcherName  string `json:"searcher_name" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	Origin        string `json:"origin" binding:"required"`
	Destination   string `json:"destination" binding:"required"`
	DepartureAt   string `json:"departure_at"`   // RFC3339 — specific time mode
	DepartureDate string `json:"departure_date"` // YYYY-MM-DD — day mode
	DepartureTime string `json:"departure_time"` // HH:MM — daily mode (any day at this time)
	Flexibility   int    `json:"flexibility"`
}

// Post creates a new ride-search alert.
// @ID       createRequest
// @Tags     requests
// @Accept   json
// @Produce  json
// @Param    body  body  handler.PostRequestBody  true  "Alert to create"
// @Success  201  {object}  domain.Request
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /requests [post]
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

// Get returns a single alert by ID. Only the owner (matching X-Phone) may read it.
// @ID       getRequest
// @Tags     requests
// @Produce  json
// @Param    id       path    string  true  "Request ID"
// @Param    X-Phone  header  string  true  "Owner phone"
// @Success  200  {object}  domain.Request
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Router   /requests/{id} [get]
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

// Delete removes an alert owned by the caller.
// @ID       deleteRequest
// @Tags     requests
// @Accept   json
// @Param    id    path  string                     true  "Request ID"
// @Param    body  body  handler.DeleteRequestBody  true  "Owner phone"
// @Success  204
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /requests/{id} [delete]
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

type offerContactBody struct {
	Phone string `json:"phone" binding:"required"`
	Name  string `json:"name"`
}

// OfferContact lets anyone share their contact with a searcher, even without a posted ride.
// @ID       offerContact
// @Tags     requests
// @Accept   json
// @Produce  json
// @Param    id    path  string                    true  "Request ID"
// @Param    body  body  handler.OfferContactBody  true  "Offerer phone and name"
// @Success  204
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /requests/{id}/offer-contact [post]
func (h *RequestHandler) OfferContact(c *gin.Context) {
	var body offerContactBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.offerContact.Execute(c.Param("id"), normalizePhone(body.Phone), body.Name)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		case errors.Is(err, usecase.ErrNameRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		case err.Error() == "request not found":
			c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

// ListContactOffers returns contact offers made for a request.
// Only the request owner (X-Phone) may retrieve this list.
// @ID       listContactOffers
// @Tags     requests
// @Produce  json
// @Param    id       path    string  true  "Request ID"
// @Param    X-Phone  header  string  true  "Request owner phone"
// @Success  200  {array}   handler.ContactOfferItem
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Router   /requests/{id}/offers [get]
func (h *RequestHandler) ListContactOffers(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	offers, err := h.getRequestContactOffers.Execute(c.Param("id"), phone)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	out := make([]ContactOfferItem, len(offers))
	for i, o := range offers {
		out[i] = ContactOfferItem{
			ID:           o.ID,
			OffererName:  o.OffererName,
			OffererPhone: o.OffererPhone,
		}
	}
	c.JSON(http.StatusOK, out)
}
