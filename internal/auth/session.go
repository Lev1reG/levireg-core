package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// HashToken is the sha256 of a session token; only this is stored.
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// NewSessionToken returns a random token and its storage hash.
func NewSessionToken() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, HashToken(token), nil
}
