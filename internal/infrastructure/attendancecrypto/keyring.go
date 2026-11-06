// Package attendancecrypto protects short-lived attendance location evidence.
package attendancecrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// Keyring encrypts location evidence with versioned AES-256-GCM keys.
// Secret bytes are supplied by infrastructure composition and are never
// persisted with the ciphertext.
type Keyring struct {
	active string
	keys   map[string]cipher.AEAD
}

func NewKeyring(active string, raw map[string][]byte) (*Keyring, error) {
	if active == "" || len(raw) == 0 {
		return nil, errors.New("attendance location encryption keys are required")
	}
	keyring := &Keyring{active: active, keys: make(map[string]cipher.AEAD, len(raw))}
	for id, key := range raw {
		if id == "" || len(key) != 32 {
			return nil, errors.New("attendance AES-256 key ID and 32-byte key are required")
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, errors.New("invalid attendance encryption key")
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, errors.New("unable to initialize attendance authenticated encryption")
		}
		keyring.keys[id] = aead
	}
	if _, ok := keyring.keys[active]; !ok {
		return nil, errors.New("active attendance encryption key is missing")
	}
	return keyring, nil
}

// Seal binds ciphertext to its opaque evidence document ID as authenticated
// associated data. A fresh nonce is prefixed to every ciphertext.
func (k *Keyring) Seal(documentID string, plaintext []byte) (string, []byte, error) {
	if k == nil || documentID == "" || len(plaintext) == 0 {
		return "", nil, errors.New("invalid attendance encryption input")
	}
	aead := k.keys[k.active]
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", nil, errors.New("unable to generate attendance encryption nonce")
	}
	return k.active, aead.Seal(nonce, nonce, plaintext, []byte(documentID)), nil
}

// Open authenticates the evidence document ID and decrypts data written by
// either the active key or a retained rotation key.
func (k *Keyring) Open(documentID, keyID string, ciphertext []byte) ([]byte, error) {
	if k == nil || documentID == "" || keyID == "" {
		return nil, errors.New("invalid attendance decryption request")
	}
	aead, ok := k.keys[keyID]
	if !ok || len(ciphertext) < aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("unknown attendance key or damaged evidence")
	}
	plaintext, err := aead.Open(nil, ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():], []byte(documentID))
	if err != nil {
		return nil, errors.New("attendance evidence authentication failed")
	}
	return plaintext, nil
}
