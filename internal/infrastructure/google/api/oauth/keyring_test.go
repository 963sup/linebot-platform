package oauth

import (
	"bytes"
	"testing"
)

func TestKeyringRoundTripAndTamper(t *testing.T) {
	_, err := NewKeyring("v1", map[string][]byte{"v1": []byte("too-short")})
	if err == nil { t.Fatal("weak encryption key accepted") }
	key := bytes.Repeat([]byte{0xA5}, 32)
	ring, err := NewKeyring("v1", map[string][]byte{"v1": key})
	if err != nil { t.Fatal(err) }
	original := []byte("sensitive-oauth-refresh-token")
	id, ciphertext, err := ring.Seal("owner-hash-a", original)
	if err != nil { t.Fatal(err) }
	if bytes.Contains(ciphertext, original) { t.Fatal("plaintext stored") }
	plain, err := ring.Open("owner-hash-a", id, ciphertext)
	if err != nil || !bytes.Equal(plain, original) { t.Fatal("decryption failed") }
	if _, err := ring.Open("owner-hash-b", id, ciphertext); err == nil { t.Fatal("record substitution accepted") }
	ciphertext[len(ciphertext)-1] ^= 0x20
	if _, err := ring.Open("owner-hash-a", id, ciphertext); err == nil { t.Fatal("tampered ciphertext accepted") }
}

func TestKeyringReadsOldVersionAfterRotation(t *testing.T) {
	older := bytes.Repeat([]byte{0xAA}, 32)
	newer := bytes.Repeat([]byte{0xBB}, 32)
	a, _ := NewKeyring("v1", map[string][]byte{"v1": older})
	keyID, sealed, err := a.Seal("stable-doc", []byte("token"))
	if err != nil { t.Fatal(err) }
	b, err := NewKeyring("v2", map[string][]byte{"v1": older, "v2": newer})
	if err != nil { t.Fatal(err) }
	plain, err := b.Open("stable-doc", keyID, sealed)
	if err != nil || string(plain) != "token" { t.Fatal("older key unreadable after rotation") }
	nextKey, _, err := b.Seal("stable-doc", []byte("new"))
	if err != nil || nextKey != "v2" { t.Fatal("new writes did not rotate key") }
}
