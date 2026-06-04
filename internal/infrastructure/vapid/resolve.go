// Package vapid resolves the server's Web Push VAPID keypair.
//
// The database (via Store) is the single runtime source of truth. Env vars are
// only a one-time bootstrap seed: if the DB has no keys but VAPID_* env vars are
// set, they are adopted into the DB once (preserving existing push subscribers,
// whose subscriptions are bound to the public key) and never read again.
package vapid

import (
	"context"
	"fmt"

	webpushlib "github.com/SherClockHolmes/webpush-go"
)

const (
	keyPublic  = "vapid_public"
	keyPrivate = "vapid_private"
	keyEmail   = "vapid_email"

	// defaultEmail is used when no contact address is supplied. Push services
	// accept it; operators can override by writing the vapid_email setting.
	defaultEmail = "mailto:admin@example.com"
)

// generateKeys is the keypair generator, overridable in tests. The underlying
// function returns (privateKey, publicKey, error) in that order.
var generateKeys = webpushlib.GenerateVAPIDKeys

// Keys is a resolved VAPID keypair plus the contact email.
type Keys struct {
	Public  string
	Private string
	Email   string
}

// Store is the minimal persistence the resolver needs. *postgres.SettingsRepo
// satisfies it.
type Store interface {
	Get(ctx context.Context, key string) (string, bool, error)
	InsertIfAbsent(ctx context.Context, key, value string) error
}

// Resolve returns the VAPID keys, provisioning and persisting them if needed.
// source is "db", "env", or "generated" for logging.
func Resolve(ctx context.Context, store Store, getenv func(string) string) (Keys, string, error) {
	// 1. Steady state: keys already in the DB.
	pub, pubOK, err := store.Get(ctx, keyPublic)
	if err != nil {
		return Keys{}, "", fmt.Errorf("read vapid_public: %w", err)
	}
	priv, privOK, err := store.Get(ctx, keyPrivate)
	if err != nil {
		return Keys{}, "", fmt.Errorf("read vapid_private: %w", err)
	}
	if pubOK && privOK {
		return Keys{Public: pub, Private: priv, Email: emailOrDefault(ctx, store)}, "db", nil
	}

	// 2/3. Provision: adopt env keys if present, else generate fresh.
	var newPub, newPriv, source string
	if ep, epriv := getenv("VAPID_PUBLIC_KEY"), getenv("VAPID_PRIVATE_KEY"); ep != "" && epriv != "" {
		newPub, newPriv, source = ep, epriv, "env"
	} else {
		gpriv, gpub, gerr := generateKeys()
		if gerr != nil {
			return Keys{}, "", fmt.Errorf("generate vapid keys: %w", gerr)
		}
		newPub, newPriv, source = gpub, gpriv, "generated"
	}

	email := getenv("VAPID_EMAIL")
	if email == "" {
		email = defaultEmail
	}

	// Persist race-safely: a concurrent web container may win the insert.
	if err := store.InsertIfAbsent(ctx, keyPublic, newPub); err != nil {
		return Keys{}, "", fmt.Errorf("persist vapid_public: %w", err)
	}
	if err := store.InsertIfAbsent(ctx, keyPrivate, newPriv); err != nil {
		return Keys{}, "", fmt.Errorf("persist vapid_private: %w", err)
	}
	if err := store.InsertIfAbsent(ctx, keyEmail, email); err != nil {
		return Keys{}, "", fmt.Errorf("persist vapid_email: %w", err)
	}

	// Re-read so a race loser returns the winner's committed values.
	finalPub, _, err := store.Get(ctx, keyPublic)
	if err != nil {
		return Keys{}, "", fmt.Errorf("reread vapid_public: %w", err)
	}
	finalPriv, _, err := store.Get(ctx, keyPrivate)
	if err != nil {
		return Keys{}, "", fmt.Errorf("reread vapid_private: %w", err)
	}
	return Keys{Public: finalPub, Private: finalPriv, Email: emailOrDefault(ctx, store)}, source, nil
}

// emailOrDefault returns the stored vapid_email, or defaultEmail when it is
// absent. The email is non-critical metadata, so a read error here degrades to
// the default rather than failing key resolution — by this point the key reads
// above have already succeeded against the same store.
func emailOrDefault(ctx context.Context, store Store) string {
	email, ok, err := store.Get(ctx, keyEmail)
	if err != nil || !ok || email == "" {
		return defaultEmail
	}
	return email
}
