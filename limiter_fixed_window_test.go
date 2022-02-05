package pacemaker

import (
	"context"
	"reflect"
	"testing"
	"time"
)

type testFixedWindowStep struct {
	// passTime represents how much time passed after this request was made
	passTime    time.Duration
	expectedTtw time.Duration
	expectedErr error
}

type testFixedWindow struct {
	name string

	capacity uint64

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
			name:      "rate limit is trigger after few ticks",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 23, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					passTime:    time.Second * 3, // 26''
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    time.Second * 6, // 32''
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    time.Second * 1, // 33''
					expectedTtw: time.Second * 1,
					expectedErr: ErrRateLimitExceeded,
				},
				{
					// 33'' no rate limit should apply
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
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
					passTime:    time.Second,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    time.Second * 9, // 2022-02-05 00:00:18
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
			name:      "rate limit is not released until new window",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowStep{
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
					expectedTtw: time.Second * 10,
					expectedErr: ErrRateLimitExceeded,
				},
				// Rate limit is still held. Moving 2 seconds and getting into next window
				{
					passTime:    time.Second * 11,
					expectedTtw: time.Second * 9,
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
		{
			name:      "missed slots are calculated well when several cycles passed",
			capacity:  2,
			rate:      Rate{Amount: 10, Unit: time.Second},
			startTime: time.Date(2022, 02, 05, 0, 0, 8, 0, time.UTC),
			steps: []testFixedWindowStep{
				{
					passTime:    0,
					expectedTtw: 0,
					expectedErr: nil,
				},
				// Rate Limit is not yet reached, and 30'' pass...
				{
					passTime:    time.Second * 30,
					expectedTtw: 0,
					expectedErr: nil,
				},
				// Force rate limit by making 3 consecutive requests
				{
					passTime:    time.Second,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    time.Second,
					expectedTtw: 0,
					expectedErr: nil,
				},
				{
					passTime:    0,
					expectedTtw: time.Second * 8, // (10 - 2)
					expectedErr: ErrRateLimitExceeded,
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
