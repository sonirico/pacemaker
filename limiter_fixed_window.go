package pacemaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

type fixedWindowStorage interface {
	Inc(ctx context.Context, args FixedWindowIncArgs) (int64, error)
	Get(ctx context.Context, window time.Time) (int64, error)
	LastWindow(ctx context.Context) (time.Time, error)
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
// FIXME: This rate limiter is not consistent across restarts, as there are no
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

func (l *FixedWindowRateLimiter) Try(ctx context.Context) (Result, error) {
	return l.try(ctx, 1)
}

func (l *FixedWindowRateLimiter) Check(ctx context.Context) (Result, error) {
	return l.check(ctx, 1)
}

// Dump returns the state of rate limit according storage. It never returns a ErrRateLimit error.
func (l *FixedWindowRateLimiter) Dump(ctx context.Context) (Result, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.fillDeadline(ctx); err != nil {
		return res(0, 0), err
	}

	now := l.clock.Now()

	ttw := l.deadline.Sub(now)

	var (
		c   int64
		err error
	)

	c, err = l.db.Get(ctx, l.deadline)

	if err != nil {
		// TODO: Make this behaviour configurable. If storage cannot be accessed, do we pass, or do we block...?
		return nores, err
	}

	free := l.capacity - c

	if free >= 0 {
		return res(0, free), nil
	}

	return res(ttw, 0), nil

}

func (l *FixedWindowRateLimiter) try(ctx context.Context, tokens int64) (Result, error) {
	tokens = l.validateTokens(tokens)
	if tokens > l.capacity {
		return nores, ErrTokensGreaterThanCapacity
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.fillDeadline(ctx); err != nil {
		return res(0, 0), err
	}

	now := l.clock.Now()

	l.process(now)

	ttw := l.deadline.Sub(now)

	if l.rateLimitReached {
		return res(ttw, 0), ErrRateLimitExceeded
	}

	var (
		c   int64
		err error
	)

	c, err = l.db.Inc(ctx, FixedWindowIncArgs{
		Window:   l.deadline,
		Tokens:   tokens,
		Capacity: l.capacity,
		TTL:      ttw,
	})

	if err != nil {
		// TODO: Make this behaviour configurable. If storage cannot be accessed, do we pass, or do we block...?
		return nores, err
	}

	free := l.capacity - c

	if free >= 0 {
		return res(0, l.capacity-c), nil
	}

	return res(ttw, 0), ErrRateLimitExceeded
}

func (l *FixedWindowRateLimiter) check(ctx context.Context, tokens int64) (Result, error) {
	tokens = l.validateTokens(tokens)
	if tokens > l.capacity {
		return nores, ErrTokensGreaterThanCapacity
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.fillDeadline(ctx); err != nil {
		return res(0, 0), err
	}

	now := l.clock.Now()

	l.process(now)

	ttw := l.deadline.Sub(now)

	if l.rateLimitReached {
		return res(ttw, 0), ErrRateLimitExceeded
	}

	var (
		c   int64
		err error
	)

	c, err = l.db.Get(ctx, l.deadline)

	if err != nil {
		// TODO: Make this behaviour configurable. If storage cannot be accessed, do we pass, or do we block...?
		return nores, err
	}

	free := l.capacity - c - tokens

	if free >= 0 {
		return res(0, l.capacity-c), nil
	}

	return res(ttw, 0), ErrRateLimitExceeded
}

func (l *FixedWindowRateLimiter) process(now time.Time) {
	dur := l.rate.Duration()

	if l.deadline.IsZero() {
		// Handle first request
		l.rateLimitReached = false
		l.deadline = now.Add(dur)
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
		missedCycles := now.Sub(l.deadline)/dur + 1
		l.deadline = l.deadline.Add(dur * missedCycles)
		l.rateLimitReached = false
	}
}

func (l *FixedWindowRateLimiter) fillDeadline(ctx context.Context) error {
	if !l.deadline.IsZero() {
		return nil
	}

	deadline, err := l.db.LastWindow(ctx)
	if err != nil {
		if errors.Is(ErrNoLastKey, err) {
			return nil
		}
		return err
	}

	l.deadline = deadline
	return nil
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

func (s *FixedWindowMemoryStorage) Inc(
	ctx context.Context,
	args FixedWindowIncArgs,
) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.deadline.Equal(args.Window) {
		s.deadline = args.Window
		s.counter = 0
	}

	s.counter += args.Tokens
	s.ttl = args.TTL

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

func (s *FixedWindowMemoryStorage) LastWindow(ctx context.Context) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.deadline, ctx.Err()
}

// NewFixedWindowMemoryStorage returns a new instance of FixedWindowMemoryStorage
func NewFixedWindowMemoryStorage() *FixedWindowMemoryStorage {
	return &FixedWindowMemoryStorage{}
}
