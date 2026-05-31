package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type FeedbackHandler struct {
	recordFeedback *usecase.RecordFeedback
}

func NewFeedbackHandler(recordFeedback *usecase.RecordFeedback) *FeedbackHandler {
	return &FeedbackHandler{recordFeedback: recordFeedback}
}

type feedbackRequest struct {
	Phone string `json:"phone" binding:"required"`
	Taken bool   `json:"taken"`
}

func (h *FeedbackHandler) Post(c *gin.Context) {
	var req feedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.recordFeedback.Execute(c.Param("id"), normalizePhone(req.Phone), req.Taken); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
