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

// sealedMagic prefixes ciphertexts produced by Seal (format v2). Data without
// it is the legacy v1 format from Encrypt (no associated data).
const sealedMagic = "ODE\x02"

// Seal encrypts plaintext under key with AES-256-GCM, authenticating aad
// (associated data) alongside it. aad binds the ciphertext to where it is
// stored - e.g. "client_sessions|bullets_enc|<row id>" - so a ciphertext
// copied into another row or column fails to open, even under the same key.
// Output layout: "ODE\x02" || nonce || ciphertext+tag.
func Seal(key, plaintext []byte, aad string) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(sealedMagic), len(sealedMagic)+aead.NonceSize()+len(plaintext)+aead.Overhead())
	copy(out, sealedMagic)
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	out = append(out, nonce...)
	return aead.Seal(out, nonce, plaintext, []byte(aad)), nil
}

// ErrLegacyCiphertext is returned by OpenSealed for data that is not in the
// row-bound v2 format.
var ErrLegacyCiphertext = errors.New("crypto: legacy (unbound) ciphertext rejected")

// Open reverses Seal, requiring the same aad. Legacy (v1, unbound) data from
// Encrypt is still accepted so existing rows and backups stay readable;
// IsSealed tells callers which rows still need re-sealing. Once every row is
// re-sealed, use OpenSealed so a v1 ciphertext can't be swapped between rows (N15).
func Open(key, data []byte, aad string) ([]byte, error) {
	if IsSealed(data) {
		if plaintext, err := openSealed(key, data, aad); err == nil {
			return plaintext, nil
		}
		// A legacy ciphertext whose random nonce happens to start with the
		// magic bytes (p = 2^-32) still opens below; a moved v2 ciphertext
		// fails both ways.
	}
	return Decrypt(key, data)
}

// OpenSealed opens only v2 (associated-data bound) ciphertexts. Legacy v1
// data is refused with ErrLegacyCiphertext: unbound, it could have been moved
// in from another row or column (the swap attack v2 binding closes).
func OpenSealed(key, data []byte, aad string) ([]byte, error) {
	if !IsSealed(data) {
		return nil, ErrLegacyCiphertext
	}
	return openSealed(key, data, aad)
}

func openSealed(key, data []byte, aad string) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	body := data[len(sealedMagic):]
	if len(body) < aead.NonceSize() {
		return nil, errors.New("crypto: sealed ciphertext too short")
	}
	nonce, ciphertext := body[:aead.NonceSize()], body[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("crypto: open sealed: %w", err)
	}
	return plaintext, nil
}

// IsSealed reports whether data is in the v2 (associated-data bound) format.
func IsSealed(data []byte) bool {
	return len(data) >= len(sealedMagic) && string(data[:len(sealedMagic)]) == sealedMagic
}
