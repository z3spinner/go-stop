// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package webpush

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	webpushlib "github.com/SherClockHolmes/webpush-go"
	"github.com/z3spinner/go-stop/internal/domain"
)

const (
	// recordOverhead is the fixed aes128gcm framing wrapped around the JSON
	// payload: an 86-byte content-coding header (16 salt + 4 record-size +
	// 1 key-id length + 65 ephemeral public key), a 1-byte padding delimiter,
	// and a 16-byte GCM tag. webpush-go emits an encrypted body of exactly
	// RecordSize bytes, and RecordSize must be >= len(payload)+recordOverhead,
	// so this is both the minimum and the exact size we request.
	recordOverhead = 103

	// truncateMargin trims a few bytes beyond the push service's reported
	// overage to absorb the ellipsis we append and any JSON re-encoding, so a
	// single retry is enough to fit.
	truncateMargin = 16

	// maxTruncateRetries bounds how many times we shrink Body and resend on a
	// 413 before giving up.
	maxTruncateRetries = 2

	// defaultTransientBackoff is the pause before retrying a transient (network
	// or 5xx) failure. Sends happen in the request path, so it is kept short.
	defaultTransientBackoff = 300 * time.Millisecond

	pushTTL = 6 * 60 * 60 // seconds the push service holds an undelivered message
)

// overageRe extracts the byte count from a push service's "Payload Too Large"
// response, e.g. "Converted buffer is too long by 1441 bytes".
var overageRe = regexp.MustCompile(`too long by (\d+) bytes`)

type WebPushNotifier struct {
	vapidPublic      string
	vapidPrivate     string
	vapidEmail       string
	transientBackoff time.Duration
}

func New(vapidPublic, vapidPrivate, vapidEmail string) *WebPushNotifier {
	return &WebPushNotifier{
		vapidPublic:      vapidPublic,
		vapidPrivate:     vapidPrivate,
		vapidEmail:       vapidEmail,
		transientBackoff: defaultTransientBackoff,
	}
}

// pushPayload is the minimal envelope the service worker needs. Title and URL
// are vital (the URL routes the click); Body carries the detail and is the only
// field we trim when a constrained device rejects the payload as too large.
type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// Send delivers msg to sub. It right-sizes the encrypted record to the actual
// payload, retries once on a transient failure (network error or 5xx), and on a
// 413 Payload Too Large shrinks the non-vital Body and resends — keeping Title
// and URL intact — until it fits or the truncation budget is exhausted.
func (n *WebPushNotifier) Send(sub domain.Subscription, msg domain.Message) error {
	body := msg.Body
	truncations := 0
	transientRetried := false

	for {
		payload, err := json.Marshal(pushPayload{Title: msg.Title, Body: body, URL: msg.URL})
		if err != nil {
			return fmt.Errorf("marshal message: %w", err)
		}

		status, respBody, err := n.sendOnce(sub, payload)

		// Network-level failure: retry the same payload once.
		if err != nil {
			if !transientRetried {
				transientRetried = true
				time.Sleep(n.transientBackoff)
				continue
			}
			return fmt.Errorf("send push notification: %w", err)
		}

		if status < 400 {
			return nil
		}

		// Payload too large for a constrained device: drop non-vital detail and
		// resend. Title and URL are preserved.
		if status == 413 && truncations < maxTruncateRetries && body != "" {
			body = truncateBody(body, parseOverage(respBody))
			truncations++
			continue
		}

		// Transient server error: retry the same payload once.
		if status >= 500 && !transientRetried {
			transientRetried = true
			time.Sleep(n.transientBackoff)
			continue
		}

		return fmt.Errorf("push service status %d host=%s body=%s",
			status, hostOf(sub.Endpoint), respBody)
	}
}

// sendOnce performs a single encrypted POST and returns the HTTP status and
// response body. A non-nil error means the request never completed (network
// level); HTTP error statuses are reported via the status return.
func (n *WebPushNotifier) sendOnce(sub domain.Subscription, payload []byte) (int, string, error) {
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
		// Ride alerts are time-sensitive, so 6h avoids waking a device hours
		// later for a ride that has already departed.
		TTL:     pushTTL,
		Urgency: webpushlib.UrgencyHigh,
		// Right-size the record to the actual payload. Left unset the library
		// pads every message to a fixed 4 KB, which constrained-device
		// subscriptions (notably some Mozilla autopush endpoints) reject with
		// a 413.
		RecordSize: uint32(len(payload)) + recordOverhead,
	})
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody), nil
}

// parseOverage returns the number of bytes a push service reported the payload
// exceeded its limit by, or 0 if the response does not contain that detail.
func parseOverage(respBody string) int {
	m := overageRe.FindStringSubmatch(respBody)
	if len(m) < 2 {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

// truncateBody shortens body by at least overageBytes (plus a small margin) so
// the re-encrypted payload fits a constrained push service. It cuts on a UTF-8
// rune boundary and appends an ellipsis. If overageBytes is 0 (the service did
// not report a size) it halves the body; if the cut would leave nothing, it
// returns an empty string.
func truncateBody(body string, overageBytes int) string {
	if overageBytes <= 0 {
		overageBytes = len(body) / 2
	}
	target := len(body) - overageBytes - truncateMargin
	if target <= 0 {
		return ""
	}
	// Back up to the start of a rune so we never split a multi-byte character.
	for target > 0 && !utf8.RuneStart(body[target]) {
		target--
	}
	if target == 0 {
		return ""
	}
	return strings.TrimRight(body[:target], " ") + "…"
}

// hostOf returns the host of a push endpoint for logging, never the full token path.
func hostOf(endpoint string) string {
	if u, err := url.Parse(endpoint); err == nil {
		return u.Host
	}
	return endpoint
}
