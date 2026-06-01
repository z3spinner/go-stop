package postgres_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func newTestCrypto(t *testing.T) *postgres.PhoneCrypto {
	t.Helper()
	// 32 bytes = "test-key-32bytes-padded-to-length" encoded as base64
	c, err := postgres.NewPhoneCryptoFromEnvKey("dGVzdC1rZXktMzJieXRlcy1mb3ItdGVzdGluZyEhISE=")
	if err != nil {
		t.Fatalf("new crypto: %v", err)
	}
	return c
}

func TestPhoneCrypto_EncryptDecrypt(t *testing.T) {
	c := newTestCrypto(t)

	phones := []string{"+33611000001", "0622000002", "+447700900123"}
	for _, phone := range phones {
		enc, err := c.Encrypt(phone)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", phone, err)
		}
		if enc == phone {
			t.Errorf("Encrypt(%q) returned plaintext unchanged", phone)
		}
		dec, err := c.Decrypt(enc)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", enc, err)
		}
		if dec != phone {
			t.Errorf("round-trip: got %q, want %q", dec, phone)
		}
	}
}

func TestPhoneCrypto_Deterministic(t *testing.T) {
	c := newTestCrypto(t)
	phone := "+33611000001"

	enc1, _ := c.Encrypt(phone)
	enc2, _ := c.Encrypt(phone)
	if enc1 != enc2 {
		t.Error("Encrypt is not deterministic — same phone should produce same ciphertext")
	}
}

func TestPhoneCrypto_DifferentPhonesDifferentCiphertext(t *testing.T) {
	c := newTestCrypto(t)
	enc1, _ := c.Encrypt("+33611000001")
	enc2, _ := c.Encrypt("+33611000002")
	if enc1 == enc2 {
		t.Error("different phones should produce different ciphertexts")
	}
}

func TestPhoneCrypto_LegacyPlaintext(t *testing.T) {
	c := newTestCrypto(t)
	// Plaintext that is not valid base64 ciphertext should pass through unchanged
	plain := "+33611000001"
	dec, err := c.Decrypt(plain)
	if err != nil {
		t.Fatalf("Decrypt legacy: %v", err)
	}
	if dec != plain {
		t.Errorf("legacy plaintext: got %q, want %q", dec, plain)
	}
}

func TestPhoneCrypto_Disabled(t *testing.T) {
	c := postgres.NoopCrypto() // no key = disabled
	phone := "+33611000001"
	enc, _ := c.Encrypt(phone)
	if enc != phone {
		t.Error("disabled crypto should return phone unchanged on Encrypt")
	}
	dec, _ := c.Decrypt(phone)
	if dec != phone {
		t.Error("disabled crypto should return phone unchanged on Decrypt")
	}
}
