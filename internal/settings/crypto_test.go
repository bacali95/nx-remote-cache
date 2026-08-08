package settings

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func testKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	enc, err := NewEncryptor(testKey(t))
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := "AKIAIOSFODNN7EXAMPLE"
	blob, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if blob == plaintext {
		t.Fatalf("ciphertext equals plaintext")
	}

	got, err := enc.Decrypt(blob)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("Decrypt = %q, want %q", got, plaintext)
	}
}

func TestEncryptNondeterministic(t *testing.T) {
	enc, err := NewEncryptor(testKey(t))
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	a, _ := enc.Encrypt("same input")
	b, _ := enc.Encrypt("same input")
	if a == b {
		t.Fatalf("two encryptions of the same plaintext produced identical ciphertext (nonce reuse?)")
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	enc1, _ := NewEncryptor(testKey(t))
	enc2, _ := NewEncryptor(testKey(t))

	blob, err := enc1.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := enc2.Decrypt(blob); err == nil {
		t.Fatalf("Decrypt with wrong key succeeded, want error")
	}
}

func TestNewEncryptorRejectsBadKey(t *testing.T) {
	if _, err := NewEncryptor("not-base64!!!"); err == nil {
		t.Fatalf("expected error for invalid base64")
	}
	shortKey := base64.StdEncoding.EncodeToString([]byte("too-short"))
	if _, err := NewEncryptor(shortKey); err == nil {
		t.Fatalf("expected error for key that isn't 32 bytes")
	}
}
