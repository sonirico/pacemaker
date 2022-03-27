package pacemaker

import (
	"context"
	"github.com/sonirico/pacemaker/internal"
	"sync"
	"time"
)

type fixedTruncatedWindowStorage interface {
	Inc(
		ctx context.Context,
		args fixedWindowStorageIncArgs,
	) (int64, error)
}

type FixedTruncatedWindowArgs struct {
	Capacity int64
	Rate     Rate

	Clock clock

	DB fixedTruncatedWindowStorage
}

// FixedTruncatedWindowRateLimiter limits how many requests can be make in a time window. This window is calculated
// by truncating the first request's time of to the limit rate in order to adjust to real time passing. E.g:
// First request time: 2022-02-05 10:23:23
// Rate limit interval: new window every 10 seconds
// First request window: from 2022-02-05 10:23:20 to 2022-02-05 10:23:30
type FixedTruncatedWindowRateLimiter struct {
	db             fixedTruncatedWindowStorage
	clock          clock
	validateTokens func(int64) int64

	mu sync.Mutex

	rate             time.Duration
	window           time.Time
	capacity         int64
	rateLimitReached bool
}

// Check returns how much time to wait to perform the request and an error indicating whether the rate limit
// was exhausted or any kind or error happened when updating the backend. Typically, you would do
//
// ttw, err := limiter.Check(ctx)
// if errors.Is(ErrRateLimitExceeded) {
// 		<-time.After(ttw) // Wait, or enqueue your request
// }
func (l *FixedTruncatedWindowRateLimiter) Check(ctx context.Context) (time.Duration, error) {
	return l.check(ctx, 1)
}

func (l *FixedTruncatedWindowRateLimiter) check(ctx context.Context, tokens int64) (time.Duration, error) {
	tokens = l.validateTokens(tokens)
	if tokens > l.capacity {
		return 0, ErrTokensGreaterThanCapacity
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()

	window := now.Truncate(l.rate)

	if !l.window.Equal(window) {
		l.rateLimitReached = false
		l.window = window
	}

	ttw := l.rate - now.Sub(window)

	if l.rateLimitReached {
		return ttw, ErrRateLimitExceeded
	}

	c, err := l.db.Inc(ctx, fixedWindowIncArgs{
		window: window,
		tokens: tokens,
		ttl:    ttw,
	})

	if err != nil {
		return 0, err
	}

	if c > l.capacity {
		l.rateLimitReached = true
		return ttw, ErrRateLimitExceeded
	}

	return 0, nil
}

func (l *FixedTruncatedWindowRateLimiter) fixedWindow() {}

// NewFixedTruncatedWindowRateLimiter returns a new instance of FixedTruncatedWindowRateLimiter from struct of args
func NewFixedTruncatedWindowRateLimiter(
	args FixedTruncatedWindowArgs,
) *FixedTruncatedWindowRateLimiter {
	return &FixedTruncatedWindowRateLimiter{
		capacity:         args.Capacity,
		rate:             args.Rate.Duration(),
		clock:            args.Clock,
		db:               args.DB,
		rateLimitReached: false,
		validateTokens:   internal.AtLeast(1),
	}
}

// FixedTruncatedWindowMemoryStorage is an in-memory storage for the rate limit state. Preferred option when testing and
// working with standalone instances of your program and do not care about it restarting and not being exactly compliant
// with the state of rate limits at the server
type FixedTruncatedWindowMemoryStorage struct {
	mu             sync.Mutex
	previousWindow time.Time
	counter        int64
	ttl            time.Duration
}

func (s *FixedTruncatedWindowMemoryStorage) Inc(
	ctx context.Context,
	args fixedWindowStorageIncArgs,
) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.previousWindow.Equal(args.Window()) {
		s.previousWindow = args.Window()
		s.counter = 0
	}

	s.counter += args.Tokens()
	s.ttl = args.TTL()

	return s.counter, ctx.Err()
}

// NewFixedTruncatedWindowMemoryStorage returns a new instance of FixedTruncatedWindowMemoryStorage
func NewFixedTruncatedWindowMemoryStorage() *FixedTruncatedWindowMemoryStorage {
	return &FixedTruncatedWindowMemoryStorage{}
}
