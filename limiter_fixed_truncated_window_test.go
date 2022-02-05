package pacemaker

import (
	"context"
	"reflect"
	"testing"
	"time"
)

type testFixedWindowTruncatedStep struct {
	passTime    time.Duration
	expectedTtw time.Duration
	expectedErr error
}

type testFixedWindowTruncated struct {
	name string

	capacity uint64

	rate Rate

	startTime time.Time

	steps []testFixedWindowTruncatedStep
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
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    0,
					expectedTtw: time.Second * 10,
					expectedErr: ErrRateLimitExceeded,
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
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
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
					passTime:    time.Second,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    time.Second * 2, // 2022-02-05 00:00:11
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
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
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				// Rate Limit is reached and 1 second passes...
				{
					passTime:    time.Second,
					expectedTtw: time.Second * 2,
					expectedErr: ErrRateLimitExceeded,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					passTime:    time.Second * 2,
					expectedTtw: time.Second,
					expectedErr: ErrRateLimitExceeded,
				},
				// Requests can be made again
				{
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
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
				DB:       newFixedTruncatedWindowMemoryStorage(),
				Rate:     test.rate,
			})

			for i, step := range test.steps {
				ttw, err := rl.Check(ctx)

				if !reflect.DeepEqual(err, step.expectedErr) {
					t.Errorf("step(%d) unexpected error, want %v, have %v",
						i+1, step.expectedErr, err)
				}

				if ttw != step.expectedTtw {
					t.Errorf("step(%d) unexpected time to wait, want %v, have %v",
						i+1, step.expectedTtw, ttw)
				}

				clock.Forward(step.passTime)
			}
		})
	}
}
