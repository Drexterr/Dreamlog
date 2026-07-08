// Package crypto implements envelope encryption for sensitive therapist data.
//
// Model: each therapist has a random 32-byte data key (DEK). All sensitive
// fields are encrypted with the DEK using AES-256-GCM. The DEK itself is
// stored wrapped (encrypted) by a master key (KEK) that lives only in the
// server environment - the database never contains a usable key, so a stolen
// DB dump or backup yields only ciphertext.
//
// This is encryption at rest with server-held keys - NOT end-to-end
// encryption. The server decrypts transiently in memory to serve the
// authenticated therapist and to run AI processing (OCR, summaries).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

const KeySize = 32 // AES-256

var ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")

// Encrypt seals plaintext with AES-256-GCM under key.
// Output layout: nonce || ciphertext+tag. A fresh random nonce is used per call.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt opens data produced by Encrypt.
func Decrypt(key, data []byte) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	if len(data) < aead.NonceSize() {
		return nil, ErrCiphertextTooShort
	}
	nonce, ciphertext := data[:aead.NonceSize()], data[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}
	return plaintext, nil
}

// NewDEK returns a fresh random 32-byte data key.
func NewDEK() ([]byte, error) {
	dek := make([]byte, KeySize)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("crypto: generate DEK: %w", err)
	}
	return dek, nil
}

// WrapDEK encrypts a data key under the master key for storage.
func WrapDEK(masterKey, dek []byte) ([]byte, error) { return Encrypt(masterKey, dek) }

// UnwrapDEK recovers a data key from its wrapped form.
func UnwrapDEK(masterKey, wrapped []byte) ([]byte, error) {
	dek, err := Decrypt(masterKey, wrapped)
	if err != nil {
		return nil, err
	}
	if len(dek) != KeySize {
		return nil, fmt.Errorf("crypto: unwrapped DEK has %d bytes, want %d", len(dek), KeySize)
	}
	return dek, nil
}

// ParseMasterKey decodes MASTER_ENCRYPTION_KEY. Accepts 64 hex chars or
// standard base64 of 32 bytes.
func ParseMasterKey(s string) ([]byte, error) {
	if b, err := hex.DecodeString(s); err == nil && len(b) == KeySize {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == KeySize {
		return b, nil
	}
	return nil, errors.New("crypto: MASTER_ENCRYPTION_KEY must be 32 bytes (64 hex chars or base64)")
}

// ResolveMasterKey returns the master key to use: the configured
// MASTER_ENCRYPTION_KEY when set (must parse to 32 bytes), otherwise a key
// derived from fallbackSecret. derived=true signals the caller to log a
// warning recommending an explicit key in production.
func ResolveMasterKey(configured, fallbackSecret string) (key []byte, derived bool, err error) {
	if configured != "" {
		key, err = ParseMasterKey(configured)
		return key, false, err
	}
	return DeriveMasterKey(fallbackSecret), true, nil
}

// DeriveMasterKey derives a 32-byte key from an existing strong secret.
// Fallback for environments where MASTER_ENCRYPTION_KEY is not set: keyed off
// a secret that is already required (the JWT secret), domain-separated so the
// derived key never equals anything used for signing.
func DeriveMasterKey(secret string) []byte {
	sum := sha256.Sum256([]byte("dreamlog-notes-encryption-v1:" + secret))
	return sum[:]
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("crypto: key has %d bytes, want %d", len(key), KeySize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}
	return aead, nil
}
