package postgres

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
)

// PhoneCrypto provides deterministic AES-256-GCM encryption for phone numbers.
//
// Deterministic means the same plaintext + key always produces the same
// ciphertext, which is required so that WHERE phone = $encrypted queries
// continue to work. The nonce is derived as HMAC-SHA256(key, plaintext)[:12]
// — equivalent to AES-SIV — so no two distinct phones share a nonce.
//
// If no key is configured the methods are no-ops (plaintext stored as-is),
// which keeps the dev stack working without configuration.
type PhoneCrypto struct {
	key []byte // 32 bytes (AES-256); nil = no encryption
}

// NoopCrypto returns a PhoneCrypto with encryption disabled (for tests/dev).
func NoopCrypto() *PhoneCrypto { return &PhoneCrypto{} }

// NewPhoneCryptoFromEnv reads PHONE_ENCRYPTION_KEY (base64-encoded 32 bytes).
// Returns a no-op crypto if the variable is unset.
func NewPhoneCryptoFromEnv() (*PhoneCrypto, error) {
	return NewPhoneCryptoFromEnvKey(os.Getenv("PHONE_ENCRYPTION_KEY"))
}

// NewPhoneCryptoFromEnvKey creates a PhoneCrypto from a base64-encoded key string.
// Returns a no-op crypto if raw is empty.
func NewPhoneCryptoFromEnvKey(raw string) (*PhoneCrypto, error) {
	if raw == "" {
		return &PhoneCrypto{}, nil
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, errors.New("PHONE_ENCRYPTION_KEY: invalid base64")
	}
	if len(key) != 32 {
		return nil, errors.New("PHONE_ENCRYPTION_KEY: must be exactly 32 bytes (256 bits)")
	}
	return &PhoneCrypto{key: key}, nil
}

// Enabled reports whether encryption is active.
func (c *PhoneCrypto) Enabled() bool { return len(c.key) == 32 }

// Encrypt returns the AES-256-GCM ciphertext of phone as a base64 string.
// Returns phone unchanged when encryption is disabled.
func (c *PhoneCrypto) Encrypt(phone string) (string, error) {
	if !c.Enabled() {
		return phone, nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := c.deriveNonce(phone)
	// Seal: nonce | ciphertext | tag
	sealed := gcm.Seal(nonce, nonce, []byte(phone), nil)
	return base64.URLEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Returns phone unchanged when encryption is
// disabled or when the value appears to be legacy plaintext.
func (c *PhoneCrypto) Decrypt(stored string) (string, error) {
	if !c.Enabled() {
		return stored, nil
	}
	data, err := base64.URLEncoding.DecodeString(stored)
	if err != nil || len(data) < 12+16 { // nonce(12) + min tag(16)
		return stored, nil // legacy plaintext — return as-is
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, data[:12], data[12:], nil)
	if err != nil {
		return stored, nil // authentication failed — likely legacy plaintext
	}
	return string(plain), nil
}

// deriveNonce computes a deterministic 12-byte nonce from HMAC-SHA256(key, phone).
func (c *PhoneCrypto) deriveNonce(phone string) []byte {
	mac := hmac.New(sha256.New, c.key)
	mac.Write([]byte(phone))
	return mac.Sum(nil)[:12]
}
