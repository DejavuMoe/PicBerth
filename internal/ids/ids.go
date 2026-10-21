// Package ids generates the random identifiers PicBerth hands out: image IDs, API token secrets,
// session IDs and the one-time setup code. Everything comes from crypto/rand.
package ids

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// alphabet is Crockford base32 in lowercase (no i, l, o, u).
const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

const (
	// ImageIDLen is 20 base32 characters = 100 bits of entropy.
	ImageIDLen = 20
	// TokenPrefix marks API token secrets so secret scanners can find them.
	TokenPrefix = "pb_"
	// TokenBodyLen is 32 base32 characters = 160 bits.
	TokenBodyLen = 32
	// TokenDisplayLen is how much of a secret is kept readable ("pb_" + 6 characters).
	TokenDisplayLen = 9
)

func randomBase32(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("ids: crypto/rand failed: " + err.Error())
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = alphabet[b&31] // 256 is a multiple of 32, so there is no modulo bias
	}
	return string(out)
}

// NewImageID returns a fresh public image ID.
func NewImageID() string { return randomBase32(ImageIDLen) }

// ValidImageID reports whether s is syntactically a canonical (lowercase) image ID.
func ValidImageID(s string) bool {
	if len(s) != ImageIDLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(alphabet, s[i]) < 0 {
			return false
		}
	}
	return true
}

// NewTokenSecret returns a new API token secret ("pb_" + 32 base32 characters).
func NewTokenSecret() string { return TokenPrefix + randomBase32(TokenBodyLen) }

// ValidTokenSecret reports whether s has the shape of a token secret.
func ValidTokenSecret(s string) bool {
	if len(s) != len(TokenPrefix)+TokenBodyLen || !strings.HasPrefix(s, TokenPrefix) {
		return false
	}
	for i := len(TokenPrefix); i < len(s); i++ {
		if strings.IndexByte(alphabet, s[i]) < 0 {
			return false
		}
	}
	return true
}

// TokenDisplay returns the non-secret prefix shown in token listings.
func TokenDisplay(secret string) string {
	if len(secret) < TokenDisplayLen {
		return secret
	}
	return secret[:TokenDisplayLen]
}

// NewSessionID returns 32 random bytes, base64url-encoded without padding.
func NewSessionID() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("ids: crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// NewSetupCode returns a one-time setup code like "7K2M-9Q4X-W8BD" (uppercase Crockford).
func NewSetupCode() string {
	s := strings.ToUpper(randomBase32(12))
	return s[0:4] + "-" + s[4:8] + "-" + s[8:12]
}

// NormalizeSetupCode upper-cases and strips separators and spaces so users can type the code loosely.
func NormalizeSetupCode(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	return strings.NewReplacer("-", "", " ", "").Replace(s)
}

// RandomHex returns n random bytes as 2n lowercase hex characters (temp file names, request IDs).
func RandomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("ids: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(buf)
}

// Hash returns the lowercase hex SHA-256 of s. Used for session IDs and token secrets, which are
// high-entropy random values, so a fast hash is appropriate.
func Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ObjectKey derives the storage key "ab/cd/<id>.<ext>" from an image ID. The key only contains [0-9a-z./].
func ObjectKey(id, ext string) string {
	return id[0:2] + "/" + id[2:4] + "/" + id + "." + ext
}
