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
	dump  testMethod = "dump"
)

func assertFixedWindowTruncatedStepEquals(
	t *testing.T,
	idx int,
	actual Result,
	actualErr error,
	expected testFixedWindowTruncatedStep,
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
					passTime:          0,
					method:            try,
					expectedTtw:       0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
				{
					passTime:          0,
					method:            try,
					expectedTtw:       0,
					expectedErr:       nil,
					expectedFreeSlots: 0,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       ErrRateLimitExceeded,
					expectedTtw:       time.Second * 10,
					expectedFreeSlots: 0,
				},
				{
					passTime:          0,
					method:            try,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
				},
				{
					method:            check,
					passTime:          0,
					expectedTtw:       time.Second * 10,
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
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
					expectedTtw:       0,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedErr:       nil,
					expectedFreeSlots: 0,
				},
				{
					method:            check,
					passTime:          time.Second,
					expectedErr:       ErrRateLimitExceeded,
					expectedTtw:       time.Second * 10,
					expectedFreeSlots: 0,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       time.Second * 9,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
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
					method:            try,
					passTime:          time.Second,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
					expectedTtw:       0,
				},
				{
					method:            try,
					passTime:          time.Second * 2, // 2022-02-05 00:00:11
					expectedTtw:       0,
					expectedFreeSlots: 0,
					expectedErr:       nil,
				},
				{
					method:            check,
					passTime:          time.Second * 7,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 7,
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
					expectedErr:       nil,
					expectedFreeSlots: 1,
					expectedTtw:       0,
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
					expectedTtw:       0,
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
					expectedErr:       nil,
					expectedFreeSlots: 1,
					expectedTtw:       0,
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
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
				},
				// Rate Limit is reached and 1 second passes...
				{
					method:            try,
					passTime:          time.Second,
					expectedTtw:       time.Second * 10,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
				},
				{
					method:            check,
					passTime:          0,
					expectedErr:       ErrRateLimitExceeded,
					expectedTtw:       time.Second * 9,
					expectedFreeSlots: 0,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					method:            try,
					passTime:          time.Second * 2,
					expectedTtw:       time.Second * 9,
					expectedErr:       ErrRateLimitExceeded,
					expectedFreeSlots: 0,
				},
				{
					method:            check,
					passTime:          time.Second * 7,
					expectedErr:       ErrRateLimitExceeded,
					expectedTtw:       time.Second * 7,
					expectedFreeSlots: 0,
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
					expectedErr:       nil,
					expectedFreeSlots: 1,
					expectedTtw:       0,
				},
			},
		},
		{
			name:      "rate limit dump",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowTruncatedStep{
				{
					method:            dump,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
					expectedTtw:       0,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            dump,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
					expectedTtw:       0,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 0,
					expectedErr:       nil,
				},
				{
					method:            dump,
					passTime:          time.Second * 2,
					expectedErr:       nil,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
				},
				{
					method:            try,
					passTime:          time.Second * 8,
					expectedTtw:       time.Second * 8,
					expectedFreeSlots: 0,
					expectedErr:       ErrRateLimitExceeded,
				},
				{
					method:            dump,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
					expectedTtw:       0,
				},
			},
		},
		{
			name:      "rate limit dump is time aware",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowTruncatedStep{
				{
					method:            dump,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
					expectedTtw:       0,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 1,
					expectedErr:       nil,
				},
				{
					method:            dump,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 1,
					expectedTtw:       0,
				},
				{
					method:            try,
					passTime:          0,
					expectedTtw:       0,
					expectedFreeSlots: 0,
					expectedErr:       nil,
				},
				{
					method:            dump,
					passTime:          time.Second * 10,
					expectedErr:       nil,
					expectedFreeSlots: 0,
					expectedTtw:       time.Second * 10,
				},
				{
					method:            dump,
					passTime:          0,
					expectedErr:       nil,
					expectedFreeSlots: 2,
					expectedTtw:       0,
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
				var (
					res Result
					err error
				)

				switch step.method {
				case try:
					res, err = rl.Try(ctx)
				case check:
					res, err = rl.Check(ctx)
				case dump:
					res, err = rl.Dump(ctx)
				default:
					panic("wrong method")
				}

				if assertFixedWindowTruncatedStepEquals(t, i+1, res, err, step) {
					clock.Forward(step.passTime)
				} else {
					t.FailNow()
				}
			}
		})
	}
}
