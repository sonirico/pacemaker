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

func assertTokenFixedWindowStepEquals(
	t *testing.T,
	idx int,
	actual Result,
	actualErr error,
	expected testTokenFixedWindowStep,
) bool {
	t.Helper()
	if !errors.Is(actualErr, expected.expectedErr) {
		t.Errorf("step(%s, %d) unexpected error, want %v, have %v",
			expected.method, idx, expected.expectedErr, actualErr)
		return false
	}

	if actual.TimeToWait != expected.expectedTtw {
		t.Errorf("expected(%s, %d) unexpected time to wait, want %v, have %v",
			expected.method, idx, expected.expectedTtw, actual.TimeToWait)
		return false
	}

	if actual.FreeSlots != expected.expectedFreeSlots {
		t.Errorf("expected(%s, %d) unexpected free slots, want %d, have %d",
			expected.method, idx, expected.expectedFreeSlots, actual.FreeSlots)
		return false
	}

	return true
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
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 50,
					expectedErr:       nil,
					requestTokens:     25,
				},
				{
					method:            try,
					forwardAfter:      0,
					expectedTtw:       0,
					expectedErr:       nil,
					requestTokens:     25, // 25
					expectedFreeSlots: 25,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 25,
					expectedErr:       nil,
					requestTokens:     25,
					expectedTtw:       0,
				},
				{
					method:            try,
					forwardAfter:      0,
					expectedTtw:       0,
					expectedErr:       nil,
					requestTokens:     24, // 49
					expectedFreeSlots: 1,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     25,
					expectedTtw:       time.Second * 10,
				},
				{
					method:            try,
					forwardAfter:      0,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     3, // 52 -> Rate limit!
					expectedFreeSlots: 0,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
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
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 20,
					expectedErr:       nil,
					expectedTtw:       0,
					requestTokens:     11,
				},
				{
					method:            try,
					forwardAfter:      0,
					expectedTtw:       0,
					expectedFreeSlots: 9,
					expectedErr:       nil,
					requestTokens:     11, // 11
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedTtw:       0,
					expectedFreeSlots: 9,
					expectedErr:       nil,
					requestTokens:     3,
				},
				{
					method:            try,
					forwardAfter:      0,
					expectedTtw:       0,
					expectedFreeSlots: 6,
					expectedErr:       nil,
					requestTokens:     3, // 14
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 6,
					expectedErr:       nil,
					requestTokens:     3,
					expectedTtw:       0,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 6,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     20,
				},
				{
					method:            try,
					requestTokens:     7, // 21 -> Rate Limit!
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
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
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:            try,
					requestTokens:     1, // 1
					forwardAfter:      time.Second,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:            try,
					requestTokens:     1,
					forwardAfter:      time.Second * 9, // now it's 2022-02-05 00:00:09
					expectedFreeSlots: 0,
					expectedTtw:       0, // TODO(@sonirico): Should this be zero?
					expectedErr:       nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:            try,
					requestTokens:     2, // 2
					expectedFreeSlots: 0,
					forwardAfter:      0,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
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
					method:            check,
					forwardAfter:      0,
					expectedTtw:       0,
					expectedFreeSlots: 10,
					expectedErr:       nil,
					requestTokens:     3,
				},
				{
					method:            try,
					requestTokens:     3, // 3
					expectedFreeSlots: 7,
					forwardAfter:      0,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 7, // TODO(@sonirico): Should this be 2?
					expectedErr:       nil,
					requestTokens:     5,
					expectedTtw:       0,
				},
				{
					method:            try,
					requestTokens:     5, // 8
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       ErrRateLimitExceeded,
					expectedTtw:       time.Second * 10,
					requestTokens:     3,
				},
				{
					method:            try,
					requestTokens:     3, // 11
					forwardAfter:      time.Second,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 9,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:            try,
					requestTokens:     3, // 11
					forwardAfter:      time.Second * 9,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 9,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 10,
					expectedTtw:       0,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:            try,
					requestTokens:     3, // 3
					forwardAfter:      0,
					expectedFreeSlots: 7,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 7,
					expectedTtw:       0,
					expectedErr:       nil,
					requestTokens:     1,
				},
			},
		},
		{
			name:      "dump just works",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testTokenFixedWindowStep{
				{
					method:            dump,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					requestTokens:     1, // 1
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            dump,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					requestTokens:     1,
					expectedFreeSlots: 0,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            dump,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
					expectedErr:       nil,
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

			var (
				r   Result
				err error
			)

			for i, step := range test.steps {
				clock.Forward(step.forwardBefore)

				switch step.method {
				case try:
					r, err = rl.Try(ctx, step.requestTokens)
				case check:
					r, err = rl.Check(ctx, step.requestTokens)
				case dump:
					r, err = rl.Dump(ctx)
				}

				if assertTokenFixedWindowStepEquals(t, i+1, r, err, step) {
					clock.Forward(step.forwardAfter)
				} else {
					t.FailNow()
				}
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
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 1,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},

				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 1,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:        try,
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
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 1,               // 1
					forwardAfter:  time.Second * 3, // 26''
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 1,               // 2
					forwardAfter:  time.Second * 6, // 32''
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 1, // 3 -> Rate Limit!
					forwardAfter:  0,
					expectedTtw:   time.Second * 1,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:            check,
					forwardAfter:      time.Second,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					// 33'' no rate limit should apply
					method:        try,
					requestTokens: 1, // 1
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
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
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 5,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 1, // 1
					forwardAfter:  time.Second,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 4,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 3,               // 4
					forwardAfter:  time.Second * 9, // 2022-02-05 00:00:18
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 5, // 5 - 1 - 3
					expectedErr:       nil,
					requestTokens:     5,
				},
				{
					method:        try,
					requestTokens: 5,
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
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
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 6,
					expectedErr:       nil,
					requestTokens:     2,
				},
				{
					method:        try,
					requestTokens: 2, // 2
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 4,
					expectedErr:       nil,
					requestTokens:     2,
				},
				{
					method:        try,
					requestTokens: 2, // 4
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     3,
				},
				// Rate Limit is reached and 1 second passes...
				{
					method:        try,
					requestTokens: 3, // 7
					forwardAfter:  time.Second,
					expectedTtw:   time.Second * 10,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     3,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					method:        try,
					requestTokens: 3, // 7
					forwardAfter:  time.Second * 11,
					expectedTtw:   time.Second * 9,
					expectedErr:   ErrRateLimitExceeded,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 6,
					expectedErr:       nil,
					requestTokens:     3,
				},
				// Requests check be made again
				{
					method:        try,
					requestTokens: 3, // 3
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
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
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 1, // 1
					forwardAfter:  0,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				// Rate Limit is not yet reached, and 30'' pass...
				{
					method:        try,
					requestTokens: 1, // 2
					forwardAfter:  time.Second * 30,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
					requestTokens:     1,
				},
				// Force rate limit by making 3 consecutive requests
				{
					method:        try,
					requestTokens: 1, // 1
					forwardAfter:  time.Second,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
					requestTokens:     1,
				},
				{
					method:        try,
					requestTokens: 1, // 2
					forwardAfter:  time.Second,
					expectedTtw:   0,
					expectedErr:   nil,
				},
				{
					method:            check,
					forwardAfter:      0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
					requestTokens:     1,
				},
				{
					method:        try,
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

				errorf := func(format string, args ...any) {
					t.Errorf("[%s]"+format, step.method, args)
				}

				switch step.method {
				case try:
					res, err := rl.Try(ctx, step.requestTokens)

					if !errors.Is(err, step.expectedErr) {
						errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if res.TimeToWait != step.expectedTtw {
						errorf("step(%d) unexpected time to wait, want %v, have %v",
							i+1, step.expectedTtw, res.TimeToWait)
					}

				case check:
					res, err := rl.Check(ctx, step.requestTokens)
					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if res.FreeSlots != step.expectedFreeSlots {
						t.Errorf("step(%d) unexpected free slots, want %d, have %d",
							i+1, step.expectedFreeSlots, res.FreeSlots)
					}
				}

				clock.Forward(step.forwardAfter)
			}
		})
	}
}
