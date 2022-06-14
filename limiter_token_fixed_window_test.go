package pacemaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testTokenFixedWindowStep struct {
	method            testMethod
	forwardAfter      time.Duration
	forwardBefore     time.Duration
	expectedTtw       time.Duration
	expectedErr       error
	expectedFreeSlots int64
	requestTokens     int64
}

type testTokenFixedWindow struct {
	name string

	capacity int64

	rate Rate

	startTime time.Time

	steps []testTokenFixedWindowStep
}

func TestNewTokenFixedWindowRateLimiter_WindowTruncated(t *testing.T) {
	tests := []testTokenFixedWindow{
		{
			name:      "start of the window reaches rate limit before first tick",
			capacity:  50,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 0, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 50,
					expectedErr:       nil,
					requestTokens:     25,
				},
				{
					method:        check,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
					requestTokens: 25, // 25
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 25,
					expectedErr:       nil,
					requestTokens:     25,
				},
				{
					method:        check,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
					requestTokens: 24, // 49
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     25,
				},
				{
					method:        check,
					forwardAfter:  0,
					expectedTtw:   time.Second * 10,
					expectedErr:   ErrRateLimitExceeded,
					requestTokens: 3, // 52 -> Rate limit!
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     25,
				},
			},
		},
		{
			name:      "6s on the middle of the window reaches rate limit before first tick",
			capacity:  20,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 6, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 20,
					expectedErr:       nil,
					requestTokens:     11,
				},
				{
					method:        check,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
					requestTokens: 11, // 11
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 9,
					expectedErr:       nil,
					requestTokens:     3,
				},
				{
					method:        check,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
					requestTokens: 3, // 14
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 6,
					expectedErr:       nil,
					requestTokens:     3,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 6,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     20,
				},
				{
					method:        check,
					requestTokens: 7, // 21 -> Rate Limit!
					forwardAfter:  0,
					expectedTtw:   time.Second * 4,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
			},
		},
		{
			name:      "rate limit is not triggered after moving to new window",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1, // 1
					forwardAfter:  time.Second,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1,               // 2
					forwardAfter:  time.Second * 2, // 2022-02-05 00:00:11
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 2, // 2
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
			},
		},
		{
			name:      "rate limit is not released after new window",
			capacity:  10,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 10,
					expectedErr:       nil,
					requestTokens:     3,
				},
				{
					method:        check,
					requestTokens: 3, // 3
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 7,
					expectedErr:       nil,
					requestTokens:     5,
				},
				{
					method:        check,
					requestTokens: 5, // 8
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     3,
				},
				// Rate Limit is reached and 1 second passes...
				{
					method:        check,
					requestTokens: 3, // 11
					forwardAfter:  time.Second,
					expectedTtw:   time.Second * 2,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					method:        check,
					requestTokens: 3, // 11
					forwardAfter:  time.Second * 2,
					expectedTtw:   time.Second,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 10,
					expectedErr:       nil,
					requestTokens:     1,
				},
				// Requests can be made again
				{
					method:        check,
					requestTokens: 3, // 3
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 7,
					expectedErr:       nil,
					requestTokens:     1,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			clock := NewMockClock(test.startTime)
			rl := NewTokenFixedWindowRateLimiter(
				NewFixedTruncatedWindowRateLimiter(FixedTruncatedWindowArgs{
					Capacity: test.capacity,
					Clock:    clock,
					DB:       NewFixedTruncatedWindowMemoryStorage(),
					Rate:     test.rate,
				}),
			)

			for i, step := range test.steps {
				clock.Forward(step.forwardBefore)

				switch step.method {
				case check:
					ttw, err := rl.Check(ctx, step.requestTokens)

					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if ttw != step.expectedTtw {
						t.Errorf("step(%d) unexpected time to wait, want %v, have %v",
							i+1, step.expectedTtw, ttw)
					}

				case can:
					free, err := rl.Can(ctx, step.requestTokens)
					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if free != step.expectedFreeSlots {
						t.Errorf("step(%d) unexpected free slots, want %d, have %d",
							i+1, step.expectedFreeSlots, free)
					}
				}

				clock.Forward(step.forwardAfter)
			}
		})
	}
}

func TestNewTokenFixedWindowRateLimiter_WindowStartsWithFirstRequest(t *testing.T) {
	tests := []testTokenFixedWindow{
		{
			name:      "start of the window reaches rate limit before first tick",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 23, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},

				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1,
					forwardAfter:  0,
					expectedTtw:   time.Second * 10,
					expectedErr:   ErrRateLimitExceeded,
				},
			},
		},
		{
			name:      "rate limit is trigger after few ticks",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 23, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1,               // 1
					forwardAfter:  time.Second * 3, // 26''
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1,               // 2
					forwardAfter:  time.Second * 6, // 32''
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1, // 3 -> Rate Limit!
					forwardAfter:  0,
					expectedTtw:   time.Second * 1,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:            can,
					forwardAfter:      time.Second,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					// 33'' no rate limit should apply
					method:        check,
					requestTokens: 1, // 1
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
			},
		},
		{
			name:      "rate limit is not triggered after moving to new window",
			capacity:  5,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 5,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1, // 1
					forwardAfter:  time.Second,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 4,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 3,               // 4
					forwardAfter:  time.Second * 9, // 2022-02-05 00:00:18
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 5, // 5 - 1 - 3
					expectedErr:       nil,
					requestTokens:     5,
				},
				{
					method:        check,
					requestTokens: 5,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     5,
				},
			},
		},
		{
			name:      "rate limit is not released until new window",
			capacity:  6,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 6,
					expectedErr:       nil,
					requestTokens:     2,
				},
				{
					method:        check,
					requestTokens: 2, // 2
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 4,
					expectedErr:       nil,
					requestTokens:     2,
				},
				{
					method:        check,
					requestTokens: 2, // 4
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     3,
				},
				// Rate Limit is reached and 1 second passes...
				{
					method:        check,
					requestTokens: 3, // 7
					forwardAfter:  time.Second,
					expectedTtw:   time.Second * 10,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     3,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					method:        check,
					requestTokens: 3, // 7
					forwardAfter:  time.Second * 11,
					expectedTtw:   time.Second * 9,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 6,
					expectedErr:       nil,
					requestTokens:     3,
				},
				// Requests can be made again
				{
					method:        check,
					requestTokens: 3, // 3
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 3,
					expectedErr:       nil,
					requestTokens:     1,
				},
			},
		},
		{
			name:      "missed slots are calculated well when several cycles passed",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1, // 1
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				// Rate Limit is not yet reached, and 30'' pass...
				{
					method:        check,
					requestTokens: 1, // 2
					forwardAfter:  time.Second * 30,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				// Force rate limit by making 3 consecutive requests
				{
					method:        check,
					requestTokens: 1, // 1
					forwardAfter:  time.Second,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1, // 2
					forwardAfter:  time.Second,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            can,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:        check,
					requestTokens: 1, // 3 -> Rate limit!
					forwardAfter:  0,
					expectedTtw:   time.Second * 8, // (10 - 2)
					expectedErr:   ErrRateLimitExceeded,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			clock := NewMockClock(test.startTime)
			rl := NewTokenFixedWindowRateLimiter(
				NewFixedWindowRateLimiter(FixedWindowArgs{
					Capacity: test.capacity,
					Clock:    clock,
					DB:       NewFixedWindowMemoryStorage(),
					Rate:     test.rate,
				}),
			)

			for i, step := range test.steps {
				clock.Forward(step.forwardBefore)

				switch step.method {
				case check:
					ttw, err := rl.Check(ctx, step.requestTokens)

					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if ttw != step.expectedTtw {
						t.Errorf("step(%d) unexpected time to wait, want %v, have %v",
							i+1, step.expectedTtw, ttw)
					}

				case can:
					free, err := rl.Can(ctx, step.requestTokens)
					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if free != step.expectedFreeSlots {
						t.Errorf("step(%d) unexpected free slots, want %d, have %d",
							i+1, step.expectedFreeSlots, free)
					}
				}

				clock.Forward(step.forwardAfter)
			}
		})
	}
}
