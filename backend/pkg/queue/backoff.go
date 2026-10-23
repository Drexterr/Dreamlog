package queue

import (
	"context"
	"time"
)

// Backoff spaces out retries after queue errors: it doubles from Min up to
// Max and resets after a success. A broken or rate-limited Redis is otherwise
// hammered every couple of seconds, which on a metered provider (Upstash)
// burns the request quota - the failure it is trying to survive.
type Backoff struct {
	Min, Max time.Duration
	cur      time.Duration
}

// NewBackoff returns a Backoff doubling from min to max.
func NewBackoff(min, max time.Duration) *Backoff {
	return &Backoff{Min: min, Max: max}
}

// Next returns the delay to wait before the next attempt and doubles the one
// after it (capped at Max).
func (b *Backoff) Next() time.Duration {
	if b.cur < b.Min {
		b.cur = b.Min
	}
	d := b.cur
	b.cur *= 2
	if b.cur > b.Max {
		b.cur = b.Max
	}
	if d > b.Max {
		d = b.Max
	}
	return d
}

// Reset starts over from Min; call it after a successful dequeue.
func (b *Backoff) Reset() { b.cur = 0 }

// Sleep waits d or until ctx is cancelled; it reports whether the full wait
// elapsed (false = ctx cancelled, the caller should stop).
func Sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
