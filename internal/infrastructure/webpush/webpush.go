package webpush

import (
	"encoding/json"
	"fmt"
	"io"

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

func (n *WebPushNotifier) Send(sub domain.Subscription, msg domain.Message) error {
	payload, err := json.Marshal(msg)
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

	resp, err := webpushlib.SendNotification(payload, s, &webpushlib.Options{
		VAPIDPublicKey:  n.vapidPublic,
		VAPIDPrivateKey: n.vapidPrivate,
		Subscriber:      n.vapidEmail,
		TTL:             86400,
		Urgency:         webpushlib.UrgencyHigh,
	})
	if err != nil {
		return fmt.Errorf("send push notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push service status %d endpoint=%s body=%s",
			resp.StatusCode, sub.Endpoint, string(body))
	}
	return nil
}
