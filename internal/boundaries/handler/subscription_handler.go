package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type SubscriptionHandler struct {
	subscribe   *usecase.Subscribe
	unsubscribe *usecase.Unsubscribe
}

func NewSubscriptionHandler(subscribe *usecase.Subscribe, unsubscribe *usecase.Unsubscribe) *SubscriptionHandler {
	return &SubscriptionHandler{subscribe: subscribe, unsubscribe: unsubscribe}
}

type subscribeRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Endpoint string `json:"endpoint" binding:"required"`
	P256DH   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}

func (h *SubscriptionHandler) Subscribe(c *gin.Context) {
	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub := domain.Subscription{
		Phone:    req.Phone,
		Endpoint: req.Endpoint,
		Keys:     domain.PushKeys{P256DH: req.P256DH, Auth: req.Auth},
	}
	if err := h.subscribe.Execute(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}

func (h *SubscriptionHandler) Unsubscribe(c *gin.Context) {
	if err := h.unsubscribe.Execute(c.Param("phone")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
