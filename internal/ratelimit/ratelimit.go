// Package ratelimit provides keyed token buckets held in memory.
package ratelimit

import (
	"math"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter is a set of token buckets sharing one rate and burst. Buckets that have fully refilled are
// indistinguishable from absent ones, so they are evicted without changing behaviour.
type Limiter struct {
	mu       sync.Mutex
	rate     float64 // tokens per second
	burst    float64
	buckets  map[string]*bucket
	maxKeys  int
	lastSwep time.Time
	now      func() time.Time
}

// New creates a limiter allowing burst events at once and rate events per second sustained.
func New(rate float64, burst int) *Limiter {
	return &Limiter{rate: rate, burst: float64(burst), buckets: map[string]*bucket{}, maxKeys: 100_000, now: time.Now}
}

// PerMinute is a convenience for "n per minute with burst n".
func PerMinute(n int) *Limiter { return New(float64(n)/60, n) }

// SetNow replaces the clock (tests).
func (l *Limiter) SetNow(f func() time.Time) {
	l.mu.Lock()
	l.now = f
	l.mu.Unlock()
}

// Configure changes rate and burst for all keys; existing buckets keep their tokens (capped at the new burst).
func (l *Limiter) Configure(rate float64, burst int) {
	l.mu.Lock()
	l.rate, l.burst = rate, float64(burst)
	l.mu.Unlock()
}

func (l *Limiter) refill(b *bucket, now time.Time) {
	if el := now.Sub(b.last).Seconds(); el > 0 {
		b.tokens = math.Min(l.burst, b.tokens+el*l.rate)
	}
	b.last = now
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
}

// Allow consumes one token for key. When it refuses, retryAfter says how long until a token is available.
func (l *Limiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	b := l.buckets[key]
	if b == nil {
		if len(l.buckets) >= l.maxKeys {
			l.evictOne()
		}
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	} else {
		l.refill(b, now)
	}
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	if l.rate <= 0 {
		return false, time.Hour
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	if wait < time.Second {
		wait = time.Second
	}
	return false, wait
}

// Refund gives back one token (for attempts that turned out not to count, e.g. a successful login).
func (l *Limiter) Refund(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if b := l.buckets[key]; b != nil {
		b.tokens = math.Min(l.burst, b.tokens+1)
	}
}

// Len returns the number of tracked keys.
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// sweep drops fully refilled buckets, at most once a minute. Caller holds mu.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSwep) < time.Minute {
		return
	}
	l.lastSwep = now
	for k, b := range l.buckets {
		l.refill(b, now)
		if b.tokens >= l.burst {
			delete(l.buckets, k)
		}
	}
}

// evictOne removes an arbitrary bucket to stay under maxKeys (map iteration order is random). Caller holds mu.
func (l *Limiter) evictOne() {
	for k := range l.buckets {
		delete(l.buckets, k)
		return
	}
}
