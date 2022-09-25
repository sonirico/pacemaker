package pacemaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type (
	testMethod string

	testFixedWindowTruncatedStep struct {
		passTime          time.Duration
		method            testMethod
		expectedTtw       time.Duration
		expectedFreeSlots int64
		expectedErr       error
	}

	testFixedWindowTruncated struct {
		name string

		capacity int64

		rate Rate

		startTime time.Time

		steps []testFixedWindowTruncatedStep
	}
)

const (
	try   testMethod = "try"
	check testMethod = "check"
)

func TestNewFixedTruncatedWindowRateLimiter(t *testing.T) {
	tests := []testFixedWindowTruncated{
		{
			name:      "start of the window reaches rate limit before first tick",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 0, 0, time.UTC),
			steps: []testFixedWindowTruncatedStep{
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
				},
				{
					passTime:    0,
					method:      try,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
				{
					passTime:    0,
					method:      try,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
				},
				{
					passTime:    0,
					method:      try,
					expectedTtw: time.Second * 10,
					expectedErr: ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
				},
			},
		},
		{
			name:      "6s on the middle of the window reaches rate limit before first tick",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 6, 0, time.UTC),
			steps: []testFixedWindowTruncatedStep{
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
				},
				{
					method:      try,
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
				{
					method:      try,
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
				},
				{
					method:      try,
					passTime:    0,
					expectedTtw: time.Second * 4,
					expectedErr: ErrRateLimitExceeded,
				},
			},
		},
		{
			name:      "rate limit is not triggered after moving to new window",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowTruncatedStep{
				{
					method:      try,
					passTime:    time.Second,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
				{
					method:      try,
					passTime:    time.Second * 2, // 2022-02-05 00:00:11
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
				},
				{
					method:      try,
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				}, {
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
			},
		},
		{
			name:      "rate limit is not released after new window",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowTruncatedStep{
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
				},
				{
					method:      try,
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
				{
					method:      try,
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				// Rate Limit is reached and 1 second passes...
				{
					method:            check,
					passTime:          0,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
				},
				{
					method:      try,
					passTime:    time.Second,
					expectedTtw: time.Second * 2,
					expectedErr: ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					method:      try,
					passTime:    time.Second * 2,
					expectedTtw: time.Second,
					expectedErr: ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
				},
				// Requests check be made again
				{
					method:      try,
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			clock := NewMockClock(test.startTime)
			rl := NewFixedTruncatedWindowRateLimiter(FixedTruncatedWindowArgs{
				Capacity: test.capacity,
				Clock:    clock,
				DB:       NewFixedTruncatedWindowMemoryStorage(),
				Rate:     test.rate,
			})

			for i, step := range test.steps {
				switch step.method {
				case try:
					res, err := rl.Try(ctx)

					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if res.TimeToWait != step.expectedTtw {
						t.Errorf("step(%d) unexpected time to wait, want %v, have %v",
							i+1, step.expectedTtw, res.TimeToWait)
					}

				case check:
					res, err := rl.Check(ctx)
					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if res.FreeSlots != step.expectedFreeSlots {
						t.Errorf("step(%d) unexpected free slots, want %d, have %d",
							i+1, step.expectedFreeSlots, res.FreeSlots)
					}
				}

				clock.Forward(step.passTime)
			}
		})
	}
}
