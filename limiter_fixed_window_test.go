package pacemaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testFixedWindowStep struct {
	method testMethod
	// passTime represents how much time passed after this request was made
	passTime          time.Duration
	expectedTtw       time.Duration
	expectedErr       error
	expectedFreeSlots int64
}

type testFixedWindow struct {
	name string

	capacity int64

	rate Rate

	startTime time.Time

	steps []testFixedWindowStep
}

func TestNewFixedWindowRateLimiter(t *testing.T) {
	tests := []testFixedWindow{
		{
			name:      "start of the window reaches rate limit before first tick",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 23, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
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
					expectedFreeSlots: 1,
					expectedErr:       nil,
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
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:      try,
					passTime:    0,
					expectedTtw: time.Second * 10,
					expectedErr: ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          time.Second * 11,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
				},
			},
		},
		{
			name:      "rate limit is trigger after few ticks",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 23, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
				},
				{
					method:      try,
					passTime:    time.Second * 3, // 26''
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:      try,
					passTime:    time.Second * 6, // 32''
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:      try,
					passTime:    time.Second * 1, // 33''
					expectedTtw: time.Second * 1,
					expectedErr: ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
				},
				{
					// 33'' no rate limit should apply
					method:      try,
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
			},
		},
		{
			name:      "rate limit is not triggered after moving to new window",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
				},
				{
					method:      try,
					passTime:    time.Second,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:      try,
					passTime:    time.Second * 9, // 2022-02-05 00:00:18
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
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
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
			},
		},
		{
			name:      "rate limit is not released until new window",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
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
					expectedFreeSlots: 1,
					expectedErr:       nil,
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
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				// Rate Limit is reached and 1 second passes...
				{
					method:      try,
					passTime:    time.Second,
					expectedTtw: time.Second * 10,
					expectedErr: ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					method:      try,
					passTime:    time.Second * 11,
					expectedTtw: time.Second * 9,
					expectedErr: ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
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
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
			},
		},
		{
			name:      "missed slots are calculated well when several cycles passed",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
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
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				// Rate Limit is not yet reached, and 30'' pass...
				{
					method:      try,
					passTime:    time.Second * 30,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedErr:       nil,
				},
				// Force rate limit by making 3 consecutive requests
				{
					method:      try,
					passTime:    time.Second,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:      try,
					passTime:    time.Second,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:      try,
					passTime:    0,
					expectedTtw: time.Second * 8, // (10 - 2)
					expectedErr: ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			clock := NewMockClock(test.startTime)
			rl := NewFixedWindowRateLimiter(FixedWindowArgs{
				Capacity: test.capacity,
				Clock:    clock,
				DB:       NewFixedWindowMemoryStorage(),
				Rate:     test.rate,
			})

			for i, step := range test.steps {
				switch step.method {
				case try:
					res, err := rl.Try(ctx)
					ttw := res.TimeToWait

					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					if ttw != step.expectedTtw {
						t.Errorf("step(%d) unexpected time to wait, want %v, have %v",
							i+1, step.expectedTtw, ttw)
					}

				case check:
					res, err := rl.Check(ctx)
					if !errors.Is(err, step.expectedErr) {
						t.Errorf("step(%d) unexpected error, want %v, have %v",
							i+1, step.expectedErr, err)
					}

					free := res.FreeSlots

					if free != step.expectedFreeSlots {
						t.Errorf("step(%d) unexpected free slots, want %d, have %d",
							i+1, step.expectedFreeSlots, free)
					}
				}

				clock.Forward(step.passTime)
			}
		})
	}
}
