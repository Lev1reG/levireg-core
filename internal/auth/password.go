package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// OWASP argon2id parameters.
const (
	argonMem  = 19456 // KiB
	argonTime = 2
	argonPar  = 1
	argonKey  = 32
	argonSalt = 16
)

var b64 = base64.RawStdEncoding

// HashPassword returns a PHC-format argon2id hash with a random salt.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSalt)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMem, argonPar, argonKey)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMem, argonTime, argonPar, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword reports whether password matches a hash made by HashPassword.
// A malformed hash never matches.
func VerifyPassword(hash, password string) bool {
	parts := strings.Split(hash, "$") // "", argon2id, v=19, m=..,t=..,p=.., salt, key
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var v, m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[2], "v=%d", &v); err != nil || v != argon2.Version {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err1 := b64.DecodeString(parts[4])
	want, err2 := b64.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(want) == 0 || m == 0 || t == 0 || p == 0 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}
