// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type InterestHandler struct {
	expressInterest    *usecase.ExpressInterest
	acceptInterest     *usecase.AcceptInterest
	getInterestContact *usecase.GetInterestContact
	cancelInterest     *usecase.CancelInterest
	interestRepo       *postgres.InterestRepo
}

func NewInterestHandler(
	expressInterest *usecase.ExpressInterest,
	acceptInterest *usecase.AcceptInterest,
	getInterestContact *usecase.GetInterestContact,
	cancelInterest *usecase.CancelInterest,
	interestRepo *postgres.InterestRepo,
) *InterestHandler {
	return &InterestHandler{
		expressInterest:    expressInterest,
		acceptInterest:     acceptInterest,
		getInterestContact: getInterestContact,
		cancelInterest:     cancelInterest,
		interestRepo:       interestRepo,
	}
}

type expressInterestRequest struct {
	Phone string `json:"phone" binding:"required"`
	Name  string `json:"name"`
}

// Express records a searcher's interest in a ride.
// @ID       expressInterest
// @Tags     interests
// @Accept   json
// @Produce  json
// @Param    id    path  string                       true  "Ride ID"
// @Param    body  body  handler.ExpressInterestBody  true  "Searcher phone and name"
// @Success  201  {object}  handler.ExpressInterestResponse
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /rides/{id}/interest [post]
func (h *InterestHandler) Express(c *gin.Context) {
	var req expressInterestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	interest, err := h.expressInterest.Execute(c.Param("id"), normalizePhone(req.Phone), strings.TrimSpace(req.Name))
	if err != nil {
		if err.Error() == "searcher cannot be the driver" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "ride not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "ride not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ExpressInterestResponse{
		ID:     interest.ID,
		Status: interest.Status,
	})
}

type acceptInterestRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// Accept lets a driver accept a searcher's interest, revealing the searcher's phone.
// @ID       acceptInterest
// @Tags     interests
// @Accept   json
// @Produce  json
// @Param    id    path  string                      true  "Interest ID"
// @Param    body  body  handler.AcceptInterestBody  true  "Driver phone"
// @Success  200  {object}  handler.AcceptInterestResponse
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Router   /interests/{id}/accept [post]
func (h *InterestHandler) Accept(c *gin.Context) {
	var req acceptInterestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	searcherPhone, err := h.acceptInterest.Execute(c.Param("id"), normalizePhone(req.Phone))
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, AcceptInterestResponse{SearcherPhone: searcherPhone})
}

// Cancel lets a searcher withdraw their own pending contact request.
// Requires the caller's X-Phone header (must be the searcher who created it).
// @ID       cancelInterest
// @Tags     interests
// @Produce  json
// @Param    id       path    string  true  "Interest ID"
// @Param    X-Phone  header  string  true  "Searcher phone"
// @Success  204  "No Content"
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Failure  409  {object}  handler.ErrorResponse
// @Router   /interests/{id} [delete]
func (h *InterestHandler) Cancel(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	err := h.cancelInterest.Execute(c.Param("id"), phone)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		case errors.Is(err, usecase.ErrNotPending):
			c.JSON(http.StatusConflict, gin.H{"error": "interest is no longer pending"})
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

// GetContact returns the contact details for an accepted/shared interest.
// Requires the caller's X-Phone header (must be a party to the interest).
// @ID       getInterestContact
// @Tags     interests
// @Produce  json
// @Param    id       path    string  true  "Interest ID"
// @Param    X-Phone  header  string  true  "Caller phone"
// @Success  200  {object}  handler.ContactInfo
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Failure  404  {object}  handler.ErrorResponse
// @Router   /interests/{id}/contact [get]
func (h *InterestHandler) GetContact(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	info, err := h.getInterestContact.Execute(c.Param("id"), phone)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ContactInfo{
		Phone:       info.Phone,
		Name:        info.Name,
		Role:        info.Role,
		Origin:      info.Origin,
		Destination: info.Destination,
		DepartureAt: info.DepartureAt,
	})
}

// ListMyRequests returns all contact requests made by the authenticated searcher.
// GET /api/interests  (X-Phone header)
// @ID       listMyInterests
// @Tags     interests
// @Produce  json
// @Param    X-Phone  header  string  true  "Searcher phone"
// @Success  200  {array}  handler.MyInterest
// @Failure  401  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /interests [get]
func (h *InterestHandler) ListMyRequests(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	interests, err := h.interestRepo.FindBySearcherPhone(phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]MyInterest, len(interests))
	for i, r := range interests {
		out[i] = MyInterest{
			ID:          r.ID,
			RideID:      r.RideID,
			Status:      r.Status,
			DriverName:  r.DriverName,
			Origin:      r.Origin,
			Destination: r.Destination,
			DepartureAt: r.DepartureAt.Format("2006-01-02T15:04:05Z"),
		}
	}
	c.JSON(http.StatusOK, out)
}
