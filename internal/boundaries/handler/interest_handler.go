package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type InterestHandler struct {
	expressInterest    *usecase.ExpressInterest
	acceptInterest     *usecase.AcceptInterest
	getInterestContact *usecase.GetInterestContact
}

func NewInterestHandler(
	expressInterest *usecase.ExpressInterest,
	acceptInterest *usecase.AcceptInterest,
	getInterestContact *usecase.GetInterestContact,
) *InterestHandler {
	return &InterestHandler{
		expressInterest:    expressInterest,
		acceptInterest:     acceptInterest,
		getInterestContact: getInterestContact,
	}
}

type expressInterestRequest struct {
	Phone string `json:"phone" binding:"required"`
	Name  string `json:"name"`
}

func (h *InterestHandler) Express(c *gin.Context) {
	var req expressInterestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	interest, err := h.expressInterest.Execute(c.Param("id"), normalizePhone(req.Phone), req.Name)
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
	c.JSON(http.StatusCreated, gin.H{
		"id":     interest.ID,
		"status": interest.Status,
	})
}

type acceptInterestRequest struct {
	Phone string `json:"phone" binding:"required"`
}

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
	c.JSON(http.StatusOK, gin.H{"searcher_phone": searcherPhone})
}

func (h *InterestHandler) GetContact(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	otherPhone, err := h.getInterestContact.Execute(c.Param("id"), phone)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"phone": otherPhone})
}
