// Package password hashes and verifies passwords with Argon2id, stored as PHC strings.
package password

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// Parameters from the OWASP Password Storage Cheat Sheet (m=19 MiB, t=2, p=1).
const (
	memoryKiB = 19456
	timeCost  = 2
	threads   = 1
	saltLen   = 16
	keyLen    = 32
)

// Upper bounds accepted when parsing a stored hash, so a corrupted database row cannot make
// verification allocate gigabytes.
const (
	maxMemoryKiB = 256 * 1024
	maxTime      = 10
	maxThreads   = 16
	maxKeyLen    = 128
)

// ErrMalformed means the stored hash is not a valid argon2id PHC string.
var ErrMalformed = errors.New("password: malformed hash")

var b64 = base64.RawStdEncoding

// Hash derives a PHC string from the password with a fresh random salt.
func Hash(pw string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, timeCost, memoryKiB, threads, keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, timeCost, threads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

type phc struct {
	memory  uint32
	time    uint32
	threads uint8
	salt    []byte
	key     []byte
}

func parsePHC(s string) (phc, error) {
	var p phc
	parts := strings.Split(s, "$")
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, key
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return p, ErrMalformed
	}
	if parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return p, ErrMalformed
	}
	var seen int
	for _, kv := range strings.Split(parts[3], ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return p, ErrMalformed
		}
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return p, ErrMalformed
		}
		switch k {
		case "m":
			p.memory = uint32(n)
			seen |= 1
		case "t":
			p.time = uint32(n)
			seen |= 2
		case "p":
			if n > maxThreads {
				return p, ErrMalformed
			}
			p.threads = uint8(n)
			seen |= 4
		default:
			return p, ErrMalformed
		}
	}
	if seen != 7 || p.memory == 0 || p.memory > maxMemoryKiB || p.time == 0 || p.time > maxTime || p.threads == 0 {
		return p, ErrMalformed
	}
	var err error
	if p.salt, err = b64.DecodeString(parts[4]); err != nil || len(p.salt) < 8 {
		return p, ErrMalformed
	}
	if p.key, err = b64.DecodeString(parts[5]); err != nil || len(p.key) < 16 || len(p.key) > maxKeyLen {
		return p, ErrMalformed
	}
	return p, nil
}

// Verify checks the password against a PHC string in constant time.
func Verify(pw, encoded string) (bool, error) {
	p, err := parsePHC(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(pw), p.salt, p.time, p.memory, p.threads, uint32(len(p.key)))
	return subtle.ConstantTimeCompare(got, p.key) == 1, nil
}

// Hasher bounds how many hashes run at once (each one allocates ~19 MiB and burns a core).
type Hasher struct {
	sem       chan struct{}
	dummyOnce sync.Once
	dummy     string
}

// NewHasher allows at most concurrency simultaneous hash operations (minimum 1).
func NewHasher(concurrency int) *Hasher {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Hasher{sem: make(chan struct{}, concurrency)}
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Hasher) release() { <-h.sem }

// Hash is the bounded version of the package-level Hash.
func (h *Hasher) Hash(ctx context.Context, pw string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()
	return Hash(pw)
}

// Verify is the bounded version of the package-level Verify.
func (h *Hasher) Verify(ctx context.Context, pw, encoded string) (bool, error) {
	if err := h.acquire(ctx); err != nil {
		return false, err
	}
	defer h.release()
	return Verify(pw, encoded)
}

// Dummy runs one verification against a throwaway hash so unknown users cost the same time as known ones.
func (h *Hasher) Dummy(ctx context.Context, pw string) error {
	if err := h.acquire(ctx); err != nil {
		return err
	}
	defer h.release()
	h.dummyOnce.Do(func() { h.dummy, _ = Hash("picberth-dummy-password") })
	if h.dummy == "" {
		return errors.New("password: dummy hash unavailable")
	}
	_, err := Verify(pw, h.dummy)
	return err
}
