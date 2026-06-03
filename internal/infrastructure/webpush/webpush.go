package webpush

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	webpushlib "github.com/SherClockHolmes/webpush-go"
	"github.com/z3spinner/go-stop/internal/domain"
)

type WebPushNotifier struct {
	vapidPublic  string
	vapidPrivate string
	vapidEmail   string
}

func New(vapidPublic, vapidPrivate, vapidEmail string) *WebPushNotifier {
	return &WebPushNotifier{
		vapidPublic:  vapidPublic,
		vapidPrivate: vapidPrivate,
		vapidEmail:   vapidEmail,
	}
}

// pushPayload is the minimal envelope the service worker needs.
// Sending only these three fields keeps the encrypted payload well under
// the 4 KB Web Push limit (the full domain.Message struct — with phone,
// origin, destination, departure_at — was exceeding it).
type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

func (n *WebPushNotifier) Send(sub domain.Subscription, msg domain.Message) error {
	payload, err := json.Marshal(pushPayload{
		Title: msg.Title,
		Body:  msg.Body,
		URL:   msg.URL,
	})
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	s := &webpushlib.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpushlib.Keys{
			P256dh: sub.Keys.P256DH,
			Auth:   sub.Keys.Auth,
		},
	}

	// webpush-go prepends "mailto:" automatically for non-HTTPS subscribers,
	// so pass the bare email address — strip the prefix if the env var includes it.
	subscriber := strings.TrimPrefix(n.vapidEmail, "mailto:")

	resp, err := webpushlib.SendNotification(payload, s, &webpushlib.Options{
		VAPIDPublicKey:  n.vapidPublic,
		VAPIDPrivateKey: n.vapidPrivate,
		Subscriber:      subscriber,
		// How long the push service holds an undelivered message for an offline
		// device before dropping it. Ride alerts are time-sensitive, so 6h avoids
		// waking a device hours later for a ride that has already departed.
		TTL:             6 * 60 * 60, // 6 hours
		Urgency:         webpushlib.UrgencyHigh,
	})
	if err != nil {
		return fmt.Errorf("send push notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		host := sub.Endpoint
		if u, err := url.Parse(sub.Endpoint); err == nil {
			host = u.Host // log only the push service host, not the device token path
		}
		return fmt.Errorf("push service status %d host=%s body=%s",
			resp.StatusCode, host, string(body))
	}
	return nil
}
