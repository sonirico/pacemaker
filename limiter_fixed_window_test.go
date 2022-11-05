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

func assertFixedWindowStepEquals(
	t *testing.T,
	idx int,
	actual Result,
	actualErr error,
	expected testFixedWindowStep,
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
					expectedTtw:       0, //time.Second * 10,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 0,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            try,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          time.Second * 11,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
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
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          time.Second * 3, // 26''
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          time.Second * 6, // 32''
					expectedFreeSlots: 0,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 1,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            try,
					passTime:          time.Second * 1, // 33''
					expectedTtw:       time.Second * 1,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					// 33'' no rate limit should apply
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
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
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          time.Second,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          time.Second * 9, // 2022-02-05 00:00:18
					expectedTtw:       0,
					expectedFreeSlots: 0,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
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
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 0,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
				},
				// Rate Limit is reached and 1 second passes...
				{
					method:            try,
					passTime:          time.Second,
					expectedTtw:       time.Second * 10,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 9,
					expectedErr:       ErrRateLimitExceeded,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					method:            try,
					passTime:          time.Second * 11,
					expectedTtw:       time.Second * 9,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				// Requests check be made again
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
			},
		},
		{
			name:      "missed slots are calculated accordingly when several cycles passed",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				// Rate Limit is not yet reached, and 30'' pass...
				{
					method:            try,
					passTime:          time.Second * 30,
					expectedFreeSlots: 0,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				// Force rate limit by making 3 consecutive requests
				{
					method:            try,
					passTime:          time.Second,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          time.Second,
					expectedFreeSlots: 0,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 8,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            try,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 8, // (10 - 2)
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            check,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 8,
					expectedErr:       ErrRateLimitExceeded,
				},
			},
		},
		{
			name:      "dump works",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					method:            dump,
					passTime:          0,
					expectedFreeSlots: 2,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            dump,
					passTime:          0,
					expectedFreeSlots: 1,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       0,
					expectedErr:       nil,
				},
				{
					method:            dump,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       0, // TODO(@sonirico): Should this be 10s?
					expectedErr:       nil,
				},
				{
					method:            try,
					passTime:          0,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            dump,
					passTime:          0,
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
			rl := NewFixedWindowRateLimiter(FixedWindowArgs{
				Capacity: test.capacity,
				Clock:    clock,
				DB:       NewFixedWindowMemoryStorage(),
				Rate:     test.rate,
			})

			for i, step := range test.steps {
				var (
					r   Result
					err error
				)
				switch step.method {
				case try:
					r, err = rl.Try(ctx)
				case check:
					r, err = rl.Check(ctx)
				case dump:
					r, err = rl.Dump(ctx)

				}

				if assertFixedWindowStepEquals(t, i+1, r, err, step) {
					clock.Forward(step.passTime)
				} else {
					t.FailNow()
				}

			}
		})
	}
}
