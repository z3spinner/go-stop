package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

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
	// Validate endpoint to prevent SSRF: must be HTTPS and a known push service host.
	if err := validatePushEndpoint(req.Endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid push endpoint"})
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

// validatePushEndpoint rejects non-HTTPS URLs and private/internal hosts.
// Web Push endpoints are always HTTPS URLs at known push service domains.
func validatePushEndpoint(endpoint string) error {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return err
	}
	if u.Scheme != "https" {
		return fmt.Errorf("push endpoint must use HTTPS")
	}
	host := strings.ToLower(u.Hostname())
	// Block loopback, link-local, and private ranges by hostname
	blocked := []string{"localhost", "127.", "10.", "172.16.", "192.168.", "169.254.", "::1", "[::"}
	for _, b := range blocked {
		if strings.HasPrefix(host, b) {
			return fmt.Errorf("push endpoint host not allowed")
		}
	}
	return nil
}

func (h *SubscriptionHandler) Unsubscribe(c *gin.Context) {
	if err := h.unsubscribe.Execute(c.Param("phone")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
