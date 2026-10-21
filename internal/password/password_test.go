package password

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestHashVerify(t *testing.T) {
	enc, err := Hash("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected PHC prefix: %s", enc)
	}
	ok, err := Verify("correct horse battery", enc)
	if err != nil || !ok {
		t.Fatalf("verify correct: ok=%v err=%v", ok, err)
	}
	ok, err = Verify("wrong", enc)
	if err != nil || ok {
		t.Fatalf("verify wrong: ok=%v err=%v", ok, err)
	}
	enc2, _ := Hash("correct horse battery")
	if enc == enc2 {
		t.Fatal("salt is not random")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	good, _ := Hash("x")
	parts := strings.Split(good, "$")
	cases := []string{
		"",
		"plaintext",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0$a2V5a2V5a2V5a2V5a2V5",
		"$argon2id$v=18$m=19456,t=2,p=1$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=19456,t=2$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=99999999,t=2,p=1$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=19456,t=0,p=1$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$" + parts[5],
		"$argon2id$v=19$m=19456,t=2,p=1$" + parts[4],
	}
	for _, c := range cases {
		if ok, err := Verify("x", c); ok || err != ErrMalformed {
			t.Errorf("Verify(%q) = %v, %v; want false, ErrMalformed", c, ok, err)
		}
	}
}

func TestHasherBoundsConcurrency(t *testing.T) {
	h := NewHasher(1)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	h.sem <- struct{}{} // occupy the only slot
	if _, err := h.Hash(ctx, "x"); err == nil {
		t.Fatal("expected context error while the semaphore is full")
	}
	<-h.sem
	enc, err := h.Hash(context.Background(), "pw")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := h.Verify(context.Background(), "pw", enc); !ok || err != nil {
		t.Fatalf("verify: %v %v", ok, err)
	}
	if err := h.Dummy(context.Background(), "whatever"); err != nil {
		t.Fatal(err)
	}
}
