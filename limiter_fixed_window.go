package pacemaker

import (
	"context"
	"sync"
	"time"
)

type fixedWindowStorage interface {
	Inc(ctx context.Context, args fixedWindowStorageIncArgs) (int64, error)
	Get(ctx context.Context, window time.Time) (int64, error)
}

type FixedWindowArgs struct {
	Capacity int64
	Rate     Rate
	Clock    clock
	DB       fixedWindowStorage
}

// FixedWindowRateLimiter limits how many requests check be make in a time window. This window is calculated
// by considering the start of the window the exact same moment the first request came. E.g:
// First request time: 2022-02-05 10:23:23
// Rate limit interval: new window every 10 seconds
// First request window: from 2022-02-05 10:23:23 to 2022-02-05 10:23:33
type FixedWindowRateLimiter struct {
	rate Rate

	clock clock

	validateTokens func(int64) int64

	deadline time.Time

	mu sync.Mutex

	db fixedWindowStorage

	capacity int64

	rateLimitReached bool
}

func (l *FixedWindowRateLimiter) Try(ctx context.Context) (time.Duration, error) {
	return l.try(ctx, 1)
}

func (l *FixedWindowRateLimiter) Check(ctx context.Context) (int64, error) {
	return l.check(ctx, 1)
}

func (l *FixedWindowRateLimiter) check(ctx context.Context, tokens int64) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()

	l.process(now)

	if l.rateLimitReached {
		return 0, ErrRateLimitExceeded
	}

	c, err := l.db.Get(ctx, l.deadline)

	if err != nil {
		// TODO: Make this behaviour configurable. If storage checknot be accessed, do we pass, or do we block...?
		return 0, err
	}

	if c >= l.capacity {
		l.rateLimitReached = true
		return 0, ErrRateLimitExceeded
	}

	free := l.capacity - c - tokens

	if free >= 0 {
		return l.capacity - c, nil
	}

	return 0, ErrRateLimitExceeded
}

func (l *FixedWindowRateLimiter) try(ctx context.Context, tokens int64) (time.Duration, error) {
	tokens = l.validateTokens(tokens)
	if tokens > l.capacity {
		return 0, ErrTokensGreaterThanCapacity
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()

	l.process(now)

	ttw := l.deadline.Sub(now)

	if l.rateLimitReached {
		return ttw, ErrRateLimitExceeded
	}

	c, err := l.db.Inc(ctx, fixedWindowIncArgs{
		window: l.deadline,
		tokens: tokens,
		ttl:    ttw,
	})

	if err != nil {
		// TODO: Make this behaviour configurable. If storage checknot be accessed, do we pass, or do we block...?
		return 0, err
	}

	if c > l.capacity {
		l.rateLimitReached = true
		return ttw, ErrRateLimitExceeded
	}

	return 0, nil
}

func (l *FixedWindowRateLimiter) process(now time.Time) {
	if l.deadline.IsZero() {
		// Handle first request
		l.rateLimitReached = false
		l.deadline = now.Add(l.rate.Duration())
	} else if !l.deadline.After(now) {
		// If deadline is before in time than now, calculate next one

		// now -> 13
		// deadline -> 23
		// ----
		// now -> 25
		// deadline -> 23
		// next deadline -> 33
		// ----
		// ....
		// now -> 56
		// deadline -> 33
		// next deadline -> 63 (33 + 3 * rate) ; 3 = (56 - 33) / 10 + 1
		missedCycles := now.Sub(l.deadline)/l.rate.Duration() + 1
		l.deadline = l.deadline.Add(l.rate.Duration() * missedCycles)
		l.rateLimitReached = false
	}
}

func (l *FixedWindowRateLimiter) fixedWindow() {}

// NewFixedWindowRateLimiter returns a new instance of FixedWindowRateLimiter from struct of args
func NewFixedWindowRateLimiter(args FixedWindowArgs) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		capacity:       args.Capacity,
		rate:           args.Rate,
		clock:          args.Clock,
		db:             args.DB,
		validateTokens: AtLeast(1),
	}
}

// FixedWindowMemoryStorage is an in-memory storage for the rate limit state. Preferred option when testing and working
// with standalone instances of your program and do not care about it restarting and not being exactly compliant with
// servers rate limits
type FixedWindowMemoryStorage struct {
	mu       sync.Mutex
	counter  int64
	deadline time.Time
	ttl      time.Duration
}

func (s *FixedWindowMemoryStorage) Inc(ctx context.Context, args fixedWindowStorageIncArgs) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.deadline.Equal(args.Window()) {
		s.deadline = args.Window()
		s.counter = 0
	}

	s.counter += args.Tokens()
	s.ttl = args.TTL()

	return s.counter, ctx.Err()
}

func (s *FixedWindowMemoryStorage) Get(ctx context.Context, window time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.deadline.Equal(window) {
		s.deadline = window
		s.counter = 0
	}

	return s.counter, ctx.Err()
}

// NewFixedWindowMemoryStorage returns a new instance of FixedWindowMemoryStorage
func NewFixedWindowMemoryStorage() *FixedWindowMemoryStorage {
	return &FixedWindowMemoryStorage{}
}
