package pacemaker

import (
	"context"
	"sync"
	"time"
)

type fixedTruncatedWindowStorage interface {
	Inc(
		ctx context.Context,
		window time.Time,
	) (uint64, error)
}

// TruncateWindow indicates whether the time window in created by truncating the first request time of arrival or if
// the first request initiates the window as is.
// TruncateWindow bool

type FixedTruncatedWindowArgs struct {
	Capacity uint64
	Rate     Rate

	Clock clock

	DB fixedTruncatedWindowStorage
}

type FixedTruncatedWindowRateLimiter struct {
	db    fixedTruncatedWindowStorage
	clock clock

	mu sync.Mutex

	rate             time.Duration
	window           time.Time
	capacity         uint64
	rateLimitReached bool
}

func (l *FixedTruncatedWindowRateLimiter) Check(ctx context.Context) (time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()

	window := now.Truncate(l.rate)

	if l.window != window {
		l.rateLimitReached = false
		l.window = window
	}

	ttw := l.rate - now.Sub(window)

	if l.rateLimitReached {
		return ttw, ErrRateLimitExceeded
	}

	c, err := l.db.Inc(ctx, window)

	if err != nil {
		return 0, err
	}

	if c > l.capacity {
		l.rateLimitReached = true
		return ttw, ErrRateLimitExceeded
	}

	return 0, nil
}

func NewFixedTruncatedWindowRateLimiter(
	args FixedTruncatedWindowArgs,
) FixedTruncatedWindowRateLimiter {
	return FixedTruncatedWindowRateLimiter{
		capacity:         args.Capacity,
		rate:             args.Rate.Duration(),
		clock:            args.Clock,
		db:               args.DB,
		rateLimitReached: false,
	}
}

type fixedTruncatedWindowMemoryStorage struct {
	mu             sync.Mutex
	previousWindow time.Time
	counter        uint64
}

func (s *fixedTruncatedWindowMemoryStorage) Inc(
	ctx context.Context,
	newWindow time.Time,
) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.previousWindow != newWindow {
		s.previousWindow = newWindow
		s.counter = 0
	}

	s.counter++

	return s.counter, ctx.Err()
}

func newFixedTruncatedWindowMemoryStorage() *fixedTruncatedWindowMemoryStorage {
	return &fixedTruncatedWindowMemoryStorage{}
}
