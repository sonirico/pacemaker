package pacemaker

import (
	"context"
	"time"
)

type fixedWindowRateLimiter interface {
	check(ctx context.Context, tokens uint64) (time.Duration, error)
	fixedWindow()
}

// TokenFixedWindowRateLimiter behaves the same as a fixed-window rate limiter. However, here allow requests to hold
// an arbitrary weight, consuming as much as weight from the capacity of the inner fixed-window rate limiter. When
// using this rate limiter, keep in mind that the `capacity` attribute of the inner rate limit means the total
// tokens usable for every window, and not the total amount of requests doable on that window.
type TokenFixedWindowRateLimiter struct {
	fixedWindowRateLimiter
}

// Check returns the amount of time to wait when the rate limit has been exceeded. The total amount of tokens consumed
// by the requests can be given as argument
func (l *TokenFixedWindowRateLimiter) Check(ctx context.Context, tokens uint64) (time.Duration, error) {
	return l.fixedWindowRateLimiter.check(ctx, tokens)
}

// NewTokenFixedWindowRateLimiter returns a new instance of TokenFixedWindowRateLimiter by receiving an already
// created fixed-window rate limiter as argument.
func NewTokenFixedWindowRateLimiter(inner fixedWindowRateLimiter) TokenFixedWindowRateLimiter {
	return TokenFixedWindowRateLimiter{fixedWindowRateLimiter: inner}
}
