package attendancecrypto

import (
	"bytes"
	"testing"
)

func TestKeyringRoundTripAndDocumentBinding(t *testing.T) {
	key := bytes.Repeat([]byte{0x41}, 32)
	ring, err := NewKeyring("v1", map[string][]byte{"v1": key})
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte(`{"latitude":25.033,"longitude":121.565}`)
	keyID, ciphertext, err := ring.Seal("punch-a", plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if keyID != "v1" || bytes.Contains(ciphertext, plaintext) {
		t.Fatalf("unsafe ciphertext key=%q value=%x", keyID, ciphertext)
	}
	got, err := ring.Open("punch-a", keyID, ciphertext)
	if err != nil || !bytes.Equal(got, plaintext) {
		t.Fatalf("open=%q err=%v", got, err)
	}
	if _, err := ring.Open("punch-b", keyID, ciphertext); err == nil {
		t.Fatal("ciphertext accepted under a different evidence document")
	}
}

func TestKeyringRetainsRotationKeys(t *testing.T) {
	oldKey := bytes.Repeat([]byte{0x11}, 32)
	oldRing, err := NewKeyring("v1", map[string][]byte{"v1": oldKey})
	if err != nil {
		t.Fatal(err)
	}
	keyID, ciphertext, err := oldRing.Seal("punch-a", []byte("evidence"))
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := NewKeyring("v2", map[string][]byte{
		"v1": oldKey,
		"v2": bytes.Repeat([]byte{0x22}, 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := rotated.Open("punch-a", keyID, ciphertext); err != nil || string(got) != "evidence" {
		t.Fatalf("rotation open=%q err=%v", got, err)
	}
	newID, _, err := rotated.Seal("punch-b", []byte("new evidence"))
	if err != nil || newID != "v2" {
		t.Fatalf("new key=%q err=%v", newID, err)
	}
}

func TestKeyringRejectsInvalidConfigurationAndTampering(t *testing.T) {
	if _, err := NewKeyring("", nil); err == nil {
		t.Fatal("empty keyring accepted")
	}
	if _, err := NewKeyring("v1", map[string][]byte{"v1": []byte("short")}); err == nil {
		t.Fatal("short key accepted")
	}
	if _, err := NewKeyring("v2", map[string][]byte{"v1": bytes.Repeat([]byte{1}, 32)}); err == nil {
		t.Fatal("missing active key accepted")
	}
	ring, err := NewKeyring("v1", map[string][]byte{"v1": bytes.Repeat([]byte{1}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	keyID, ciphertext, err := ring.Seal("punch", []byte("evidence"))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[len(ciphertext)-1] ^= 0x20
	if _, err := ring.Open("punch", keyID, ciphertext); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
}
