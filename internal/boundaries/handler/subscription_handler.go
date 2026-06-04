package handler

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type SubscriptionHandler struct {
	subscribe    *usecase.Subscribe
	unsubscribe  *usecase.Unsubscribe
	sendTestPush *usecase.SendTestPush
}

func NewSubscriptionHandler(subscribe *usecase.Subscribe, unsubscribe *usecase.Unsubscribe, sendTestPush *usecase.SendTestPush) *SubscriptionHandler {
	return &SubscriptionHandler{subscribe: subscribe, unsubscribe: unsubscribe, sendTestPush: sendTestPush}
}

type testPushRequest struct {
	Phone string `json:"phone" binding:"required"`
	Lang  string `json:"lang"`
}

// TestPush sends a test notification to all of the caller's registered devices.
// @ID       testSubscription
// @Tags     subscriptions
// @Accept   json
// @Produce  json
// @Param    body  body  handler.TestPushBody  true  "Phone to test"
// @Success  200  {object}  handler.TestPushResponse
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /subscriptions/test [post]
func (h *SubscriptionHandler) TestPush(c *gin.Context) {
	var req testPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, err := h.sendTestPush.Execute(normalizePhone(req.Phone), req.Lang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, TestPushResponse{Sent: n})
}

type subscribeRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Endpoint string `json:"endpoint" binding:"required"`
	P256DH   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}

// Subscribe registers (or updates) a Web Push subscription for a phone.
// @ID       upsertSubscription
// @Tags     subscriptions
// @Accept   json
// @Param    body  body  handler.SubscriptionBody  true  "Push subscription"
// @Success  201
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /subscriptions [post]
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
		Phone:    normalizePhone(req.Phone),
		Endpoint: req.Endpoint,
		Keys:     domain.PushKeys{P256DH: req.P256DH, Auth: req.Auth},
	}
	if err := h.subscribe.Execute(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}

// knownPushServiceSuffixes is an allowlist of known Web Push service domains.
// A push endpoint hostname must equal one of these or end with ".<suffix>".
// Covers Chrome (FCM), Firefox (Mozilla), Safari/iOS (Apple), Edge (Windows).
var knownPushServiceSuffixes = []string{
	"fcm.googleapis.com",
	"push.services.mozilla.com",
	"push.apple.com",
	"notify.windows.com",
	"pushpad.xyz",
	"onesignal.com",
}

// validatePushEndpoint rejects endpoints that are not HTTPS, not from a known
// push service, or that resolve to private/loopback addresses.
// A prefix denylist is bypassable via decimal/hex IPs and DNS rebinding;
// an allowlist + resolved-IP check is the correct defence.
func validatePushEndpoint(endpoint string) error {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("push endpoint must use HTTPS")
	}

	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return fmt.Errorf("push endpoint has no host")
	}

	allowed := false
	for _, suffix := range knownPushServiceSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("push endpoint host is not a known push service")
	}

	// Resolve and validate IPs to defeat DNS rebinding / creative hostname encoding.
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("could not resolve push endpoint host: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("push endpoint resolves to a non-public address")
		}
	}
	return nil
}

// Unsubscribe removes all Web Push subscriptions for a phone.
// @ID       removeSubscription
// @Tags     subscriptions
// @Param    phone  path  string  true  "Phone"
// @Success  204
// @Failure  500  {object}  handler.ErrorResponse
// @Router   /subscriptions/{phone} [delete]
func (h *SubscriptionHandler) Unsubscribe(c *gin.Context) {
	if err := h.unsubscribe.Execute(normalizePhone(c.Param("phone"))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
