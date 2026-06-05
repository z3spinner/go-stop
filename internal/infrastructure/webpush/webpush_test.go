// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package webpush

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	webpushlib "github.com/SherClockHolmes/webpush-go"
	"github.com/z3spinner/go-stop/internal/domain"
)

// mozilla413Body mirrors the real "Payload Too Large" response observed in
// production from updates.push.services.mozilla.com.
const mozilla413Body = `{"code":413,"errno":104,"error":"Payload Too Large",` +
	`"message":"This message is intended for a constrained device and is limited ` +
	`in size. Converted buffer is too long by 1441 bytes",` +
	`"more_info":"http://autopush.readthedocs.io/en/latest/http.html#error-codes"}`

// newTestNotifier builds a notifier with real VAPID keys and no backoff so
// transient-retry tests don't sleep.
func newTestNotifier(t *testing.T) *WebPushNotifier {
	t.Helper()
	priv, pub, err := webpushlib.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate vapid keys: %v", err)
	}
	n := New(pub, priv, "test@example.com")
	n.transientBackoff = 0
	return n
}

// testSub returns a subscription with valid P-256 keys pointing at endpoint.
func testSub(t *testing.T, endpoint string) domain.Subscription {
	t.Helper()
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate p256 key: %v", err)
	}
	// Bytes() returns the 65-byte uncompressed point the library's Unmarshal expects.
	pub := privKey.PublicKey().Bytes()
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatalf("read auth: %v", err)
	}
	return domain.Subscription{
		Endpoint: endpoint,
		Keys: domain.PushKeys{
			P256DH: base64.RawURLEncoding.EncodeToString(pub),
			Auth:   base64.RawURLEncoding.EncodeToString(auth),
		},
	}
}

func TestSendRightSizesRecordToPayload(t *testing.T) {
	var gotLen int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotLen = len(b)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	n := newTestNotifier(t)
	msg := domain.Message{Title: "Ride available!", Body: "Alice is driving from A to B", URL: "/rides/123"}
	if err := n.Send(testSub(t, srv.URL), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	payload, _ := json.Marshal(pushPayload{Title: msg.Title, Body: msg.Body, URL: msg.URL})
	want := len(payload) + recordOverhead
	if gotLen != want {
		t.Errorf("encrypted body = %d bytes, want %d", gotLen, want)
	}
	if gotLen >= 4096 {
		t.Errorf("body not right-sized: %d bytes (still padded toward 4096)", gotLen)
	}
}

func TestSend413TruncatesBodyAndRetries(t *testing.T) {
	var lens []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		lens = append(lens, len(b))
		if len(lens) == 1 {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_, _ = io.WriteString(w, mozilla413Body)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	n := newTestNotifier(t)
	msg := domain.Message{Title: "Ride available!", Body: strings.Repeat("x", 2000), URL: "/rides/123"}
	if err := n.Send(testSub(t, srv.URL), msg); err != nil {
		t.Fatalf("Send should succeed after truncation: %v", err)
	}
	if len(lens) != 2 {
		t.Fatalf("expected 2 send attempts, got %d", len(lens))
	}
	if lens[1] >= lens[0] {
		t.Errorf("retry payload (%d) not smaller than original (%d)", lens[1], lens[0])
	}
}

func TestSend413GivesUpAfterMaxTruncations(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		calls++
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = io.WriteString(w, mozilla413Body)
	}))
	defer srv.Close()

	n := newTestNotifier(t)
	msg := domain.Message{Title: "T", Body: strings.Repeat("x", 2000), URL: "/u"}
	err := n.Send(testSub(t, srv.URL), msg)
	if err == nil {
		t.Fatal("expected error when 413 persists")
	}
	if !strings.Contains(err.Error(), "413") {
		t.Errorf("error should report 413, got: %v", err)
	}
	// initial attempt + maxTruncateRetries truncated attempts
	if calls != maxTruncateRetries+1 {
		t.Errorf("expected %d attempts, got %d", maxTruncateRetries+1, calls)
	}
}

func TestSendRetriesOnceOnServerError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	n := newTestNotifier(t)
	msg := domain.Message{Title: "T", Body: "b", URL: "/u"}
	if err := n.Send(testSub(t, srv.URL), msg); err != nil {
		t.Fatalf("Send should succeed after transient retry: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 attempts (one retry), got %d", calls)
	}
}

func TestSendDoesNotRetryOn403(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		calls++
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"reason":"BadJwtToken"}`)
	}))
	defer srv.Close()

	n := newTestNotifier(t)
	msg := domain.Message{Title: "T", Body: "b", URL: "/u"}
	err := n.Send(testSub(t, srv.URL), msg)
	if err == nil {
		t.Fatal("expected error on 403")
	}
	if calls != 1 {
		t.Errorf("403 must not retry, got %d attempts", calls)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should report 403, got: %v", err)
	}
}

// TestSendPreserves410ErrorString guards the contract notify.go relies on:
// a 410 must surface in the returned error so the caller can prune the sub.
func TestSendPreserves410ErrorString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusGone)
		_, _ = io.WriteString(w, "push subscription has unsubscribed or expired.")
	}))
	defer srv.Close()

	n := newTestNotifier(t)
	err := n.Send(testSub(t, srv.URL), domain.Message{Title: "T", Body: "b", URL: "/u"})
	if err == nil || !strings.Contains(err.Error(), "410") {
		t.Errorf("expected error containing 410, got: %v", err)
	}
}

func TestTruncateBodyIsRuneSafeAndEllipsizes(t *testing.T) {
	body := strings.Repeat("é", 100) // 200 bytes, 2-byte runes
	out := truncateBody(body, 50)
	if !utf8.ValidString(out) {
		t.Errorf("truncated body is not valid UTF-8: %q", out)
	}
	if !strings.HasSuffix(out, "…") {
		t.Errorf("truncated body should end with ellipsis, got: %q", out)
	}
	if len(out) >= len(body) {
		t.Errorf("truncated body (%d) not shorter than original (%d)", len(out), len(body))
	}
}

func TestTruncateBodyEmptyWhenOverageExceedsBody(t *testing.T) {
	if got := truncateBody("short", 1000); got != "" {
		t.Errorf("expected empty body when overage exceeds length, got %q", got)
	}
}

func TestTruncateBodyHalvesWhenOverageUnknown(t *testing.T) {
	body := strings.Repeat("a", 100)
	out := truncateBody(body, 0)
	if len(out) >= len(body) {
		t.Errorf("expected halving fallback to shorten, got len %d", len(out))
	}
}
