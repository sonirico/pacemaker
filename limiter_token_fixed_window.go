package pacemaker

import (
	"context"
)

type (
	rateLimit interface {
		Dump(ctx context.Context) (Result, error)
	}

	fixedWindowRateLimiter interface {
		rateLimit
		try(ctx context.Context, tokens int64) (Result, error)
		check(ctx context.Context, tokens int64) (Result, error)
		fixedWindow()
	}
)

// TokenFixedWindowRateLimiter behaves the same as a fixed-window rate limiter. However, it allows requests to hold
// an arbitrary weight, consuming as much as weight from the capacity of the inner fixed-window rate limiter. When
// using this rate limiter, keep in mind that the `capacity` attribute of the inner rate limit means the total
// tokens usable for every window, and not the total amount of requests doable on that window.
type TokenFixedWindowRateLimiter struct {
	inner fixedWindowRateLimiter
}

// Try returns the amount of time to wait when the rate limit has been exceeded. The total amount of tokens consumed
// by the requests check be given as argument
func (l *TokenFixedWindowRateLimiter) Try(ctx context.Context, tokens int64) (Result, error) {
	return l.inner.try(ctx, tokens)
}

// Check returns whether further requests check be made by returning the number of free slots
func (l *TokenFixedWindowRateLimiter) Check(ctx context.Context, tokens int64) (Result, error) {
	return l.inner.check(ctx, tokens)
}

// Dump returns the actual rate limit state according to data stores
func (l *TokenFixedWindowRateLimiter) Dump(ctx context.Context) (Result, error) {
	return l.inner.Dump(ctx)
}

// NewTokenFixedWindowRateLimiter returns a new instance of TokenFixedWindowRateLimiter by receiving an already
// created fixed-window rate limiter as argument.
func NewTokenFixedWindowRateLimiter(inner fixedWindowRateLimiter) TokenFixedWindowRateLimiter {
	return TokenFixedWindowRateLimiter{inner: inner}
}
