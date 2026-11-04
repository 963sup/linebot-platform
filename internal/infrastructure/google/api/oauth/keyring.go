package oauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// Keyring provides authenticated encryption for OAuth state and credentials.
// Inject keys from a protected secret source, never from the repository.
type Keyring struct {
	active string
	keys map[string]cipher.AEAD
}

func NewKeyring(active string, raw map[string][]byte) (*Keyring, error) {
	if active == "" || len(raw) == 0 {
		return nil, errors.New("OAuth encryption keys are required")
	}
	r := &Keyring{active: active, keys: make(map[string]cipher.AEAD, len(raw))}
	for id, key := range raw {
		if id == "" || len(key) != 32 {
			return nil, errors.New("OAuth AES-256 key ID and 32-byte key are required")
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, errors.New("invalid OAuth encryption key")
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, errors.New("unable to initialize OAuth authenticated encryption")
		}
		r.keys[id] = aead
	}
	if _, ok := r.keys[active]; !ok {
		return nil, errors.New("active OAuth encryption key missing")
	}
	return r, nil
}

// Seal uses a fresh random nonce and binds ciphertext to an opaque Firestore
// document ID. The returned key ID allows later key rotation.
func (r *Keyring) Seal(documentID string, plaintext []byte) (string, []byte, error) {
	if r == nil || documentID == "" || len(plaintext) == 0 {
		return "", nil, errors.New("invalid OAuth encryption input")
	}
	aead := r.keys[r.active]
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", nil, errors.New("unable to generate encryption nonce")
	}
	ciphertext := aead.Seal(nonce, nonce, plaintext, []byte(documentID))
	return r.active, ciphertext, nil
}

func (r *Keyring) Open(documentID, keyID string, ciphertext []byte) ([]byte, error) {
	if r == nil || documentID == "" || keyID == "" {
		return nil, errors.New("invalid OAuth decryption request")
	}
	aead, ok := r.keys[keyID]
	if !ok || len(ciphertext) < aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("unknown OAuth key or damaged record")
	}
	plaintext, err := aead.Open(nil, ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():], []byte(documentID))
	if err != nil {
		return nil, errors.New("OAuth ciphertext authentication failed")
	}
	return plaintext, nil
}
