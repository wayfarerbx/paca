// Package secret provides AES-256-GCM symmetric encryption for sensitive
// values stored at rest (e.g. GitHub personal access tokens, webhook secrets).
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// ErrInvalidKeyLength is returned when the supplied key is not 32 bytes.
var ErrInvalidKeyLength = errors.New("secret: encryption key must be exactly 32 bytes (AES-256)")

// Encryptor encrypts and decrypts short strings using AES-256-GCM.
// A fresh random nonce is generated on every Encrypt call, so repeated
// calls with the same plaintext produce different ciphertexts.
type Encryptor struct {
	key []byte
}

// NewEncryptor creates an Encryptor from a 32-byte key.
// Use encoding.Hex or encoding/base64 to decode a stored key before passing
// it here; the key must be exactly 32 bytes.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}
	k := make([]byte, 32)
	copy(k, key)
	return &Encryptor{key: k}, nil
}

// Encrypt encrypts plaintext and returns a base64-encoded string consisting of
// the random nonce followed by the GCM ciphertext+tag.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("secret: create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("secret: create GCM wrapper: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("secret: generate nonce: %w", err)
	}

	// gcm.Seal appends ciphertext+tag to nonce, giving: [nonce | ciphertext | tag]
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts a base64-encoded ciphertext produced by Encrypt.
func (e *Encryptor) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("secret: base64 decode: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("secret: create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("secret: create GCM wrapper: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("secret: ciphertext is too short to contain a nonce")
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("secret: GCM open (authenticate+decrypt): %w", err)
	}

	return string(plaintext), nil
}
