package ratelimit

import (
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestBurstAndRefill(t *testing.T) {
	c := &fakeClock{t: time.Unix(1000, 0)}
	l := PerMinute(5)
	l.SetNow(c.now)
	for i := 0; i < 5; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d refused inside burst", i)
		}
	}
	ok, retry := l.Allow("a")
	if ok {
		t.Fatal("6th request allowed")
	}
	if retry < time.Second || retry > 13*time.Second {
		t.Fatalf("retryAfter = %v", retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("independent key refused")
	}
	c.advance(12 * time.Second) // one token at 5/min
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("token not refilled")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("second request after one refill allowed")
	}
}

func TestRefund(t *testing.T) {
	c := &fakeClock{t: time.Unix(1000, 0)}
	l := New(0.001, 1)
	l.SetNow(c.now)
	l.Allow("k")
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("expected refusal")
	}
	l.Refund("k")
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("refund did not restore token")
	}
	l.Refund("k")
	l.Refund("k")
	l.Allow("k")
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("refund exceeded burst")
	}
}

func TestIdleEviction(t *testing.T) {
	c := &fakeClock{t: time.Unix(1000, 0)}
	l := PerMinute(10)
	l.SetNow(c.now)
	for _, k := range []string{"a", "b", "c"} {
		l.Allow(k)
	}
	if l.Len() != 3 {
		t.Fatalf("len %d", l.Len())
	}
	c.advance(5 * time.Minute)
	l.Allow("d") // triggers the sweep
	if l.Len() != 1 {
		t.Fatalf("idle buckets not evicted: len=%d", l.Len())
	}
}

func TestMaxKeys(t *testing.T) {
	l := PerMinute(1)
	l.maxKeys = 10
	for i := 0; i < 100; i++ {
		l.Allow(string(rune('a'+i%26)) + string(rune('a'+i/26)))
	}
	if l.Len() > 10 {
		t.Fatalf("len %d exceeds cap", l.Len())
	}
}

func TestConfigure(t *testing.T) {
	c := &fakeClock{t: time.Unix(1000, 0)}
	l := PerMinute(60)
	l.SetNow(c.now)
	l.Allow("k")
	l.Configure(1.0/60, 1)
	c.advance(time.Second)
	if ok, _ := l.Allow("k"); !ok { // tokens capped to 1 and one available
		t.Fatal("expected one token after reconfigure")
	}
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("burst not capped")
	}
}
