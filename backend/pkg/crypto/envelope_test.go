package crypto

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key, err := NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("client reported better sleep this week — नोट्स भी हिंदी में")

	ct, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, plaintext) {
		t.Fatal("ciphertext contains plaintext")
	}

	got, err := Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round trip mismatch: got %q", got)
	}
}

func TestEncrypt_UniqueNonces(t *testing.T) {
	key := testKey(t)
	a, _ := Encrypt(key, []byte("same input"))
	b, _ := Encrypt(key, []byte("same input"))
	if bytes.Equal(a, b) {
		t.Fatal("two encryptions of the same plaintext produced identical output (nonce reuse)")
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	ct, _ := Encrypt(testKey(t), []byte("secret"))
	if _, err := Decrypt(testKey(t), ct); err == nil {
		t.Fatal("decrypt with wrong key succeeded")
	}
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	key := testKey(t)
	ct, _ := Encrypt(key, []byte("secret"))
	ct[len(ct)-1] ^= 0xFF
	if _, err := Decrypt(key, ct); err == nil {
		t.Fatal("decrypt of tampered ciphertext succeeded")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	if _, err := Decrypt(testKey(t), []byte{1, 2, 3}); err == nil {
		t.Fatal("short ciphertext accepted")
	}
}

func TestWrapUnwrapDEK(t *testing.T) {
	master := testKey(t)
	dek := testKey(t)

	wrapped, err := WrapDEK(master, dek)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(wrapped, dek) {
		t.Fatal("wrapped DEK contains raw DEK")
	}

	got, err := UnwrapDEK(master, wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatal("unwrapped DEK mismatch")
	}
}

func TestUnwrapDEK_WrongMasterFails(t *testing.T) {
	wrapped, _ := WrapDEK(testKey(t), testKey(t))
	if _, err := UnwrapDEK(testKey(t), wrapped); err == nil {
		t.Fatal("unwrap with wrong master key succeeded")
	}
}

func TestParseMasterKey(t *testing.T) {
	raw := testKey(t)

	if got, err := ParseMasterKey(hex.EncodeToString(raw)); err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("hex parse failed: %v", err)
	}
	if got, err := ParseMasterKey(base64.StdEncoding.EncodeToString(raw)); err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("base64 parse failed: %v", err)
	}
	for _, bad := range []string{"", "short", hex.EncodeToString(raw[:16])} {
		if _, err := ParseMasterKey(bad); err == nil {
			t.Fatalf("invalid key %q accepted", bad)
		}
	}
}

func TestResolveMasterKey(t *testing.T) {
	raw := testKey(t)
	configured := hex.EncodeToString(raw)

	key, derived, err := ResolveMasterKey(configured, "jwt-secret")
	if err != nil || derived || !bytes.Equal(key, raw) {
		t.Fatalf("configured key not used: derived=%v err=%v", derived, err)
	}

	key, derived, err = ResolveMasterKey("", "jwt-secret")
	if err != nil || !derived || len(key) != KeySize {
		t.Fatalf("fallback derivation failed: derived=%v err=%v", derived, err)
	}
	// Deterministic and domain-separated.
	key2, _, _ := ResolveMasterKey("", "jwt-secret")
	if !bytes.Equal(key, key2) {
		t.Fatal("derived key not deterministic")
	}
	other, _, _ := ResolveMasterKey("", "other-secret")
	if bytes.Equal(key, other) {
		t.Fatal("different secrets derived the same key")
	}

	if _, _, err := ResolveMasterKey("not-a-key", "jwt-secret"); err == nil {
		t.Fatal("invalid configured key accepted")
	}
}
