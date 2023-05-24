package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
	"github.com/sonirico/pacemaker"
)

var (
	db *redis.Client
)

func TestMain(t *testing.M) {
	var pool *dockertest.Pool
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not spawn docker pool due to %s", err)
	}

	resource, err := pool.Run("redis", "7-alpine", nil)
	if err != nil {
		log.Fatalf("could not create resource due to %s", err)
	}

	fmt.Println("spawning redis container")

	if err := pool.Retry(func() error {
		db = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("localhost:%s", resource.GetPort("6379/tcp")),
		})

		fmt.Println("redis ping")

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		return db.Ping(ctx).Err()
	}); err != nil {
		log.Fatalf("cannot connect to docker due to %s", err)
	}

	fmt.Println("redis is up and running")

	returnCode := t.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("cannot purge due to %s", err)
	}

	os.Exit(returnCode)
}

func TestFixedWindow_RunOk(t *testing.T) {
	if err := db.Ping(context.Background()).Err(); err != nil {
		t.Errorf("test has failed, expected redis to be running, have error: %v", err)
	}

	opts := pacemaker.FixedWindowArgs{
		Capacity: 100,
		Rate: pacemaker.Rate{
			Amount: 1,
			Unit:   time.Minute,
		},
		Clock: pacemaker.NewClock(),
		DB: pacemaker.NewFixedWindowRedisStorage(
			db,
			pacemaker.FixedWindowRedisStorageOpts{
				Prefix: "pacemaker|run_ok",
			},
		),
	}

	limiter := pacemaker.NewFixedWindowRateLimiter(opts)

	ctx := context.Background()

	_, err := limiter.Try(ctx)
	assertNoError(t, err)

	state, err := limiter.Dump(ctx)
	assertNoError(t, err)

	assertFreeSlots(t, 99, state.FreeSlots)
}

func TestFixedWindow_ShouldMaintainState(t *testing.T) {
	if err := db.Ping(context.Background()).Err(); err != nil {
		t.Errorf("test has failed, expected redis to be running, have error: %v", err)
	}

	ctx := context.Background()
	opts := pacemaker.FixedWindowArgs{
		Capacity: 100,
		Rate: pacemaker.Rate{
			Amount: 1,
			Unit:   time.Minute,
		},
		Clock: pacemaker.NewClock(),
		DB: pacemaker.NewFixedWindowRedisStorage(
			db,
			pacemaker.FixedWindowRedisStorageOpts{
				Prefix: "pacemaker|run_should_maintain_state",
			},
		),
	}

	limiter := pacemaker.NewFixedWindowRateLimiter(opts)

	_, err := limiter.Try(ctx)
	assertNoError(t, err)

	state, err := limiter.Dump(ctx)
	assertNoError(t, err)

	assertFreeSlots(t, 99, state.FreeSlots)

	limiter = pacemaker.NewFixedWindowRateLimiter(opts)

	_, err = limiter.Try(ctx)
	assertNoError(t, err)

	state, err = limiter.Dump(ctx)
	assertNoError(t, err)

	assertFreeSlots(t, 98, state.FreeSlots)
}

func TestFixedWindow_SubMinute_CapacityGreaterThanOne_RunOk(t *testing.T) {
	if err := db.Ping(context.Background()).Err(); err != nil {
		t.Errorf("test has failed, expected redis to be running, have error: %v", err)
	}

	opts := pacemaker.FixedWindowArgs{
		Capacity: 3,
		Rate: pacemaker.Rate{
			Amount: 3,
			Unit:   time.Second,
		},
		Clock: pacemaker.NewClock(),
		DB: pacemaker.NewFixedWindowRedisStorage(
			db,
			pacemaker.FixedWindowRedisStorageOpts{
				Prefix: "pacemaker|fixed-window|cap-3",
			},
		),
	}

	limiter := pacemaker.NewFixedWindowRateLimiter(opts)

	ctx := context.Background()

	_, err := limiter.Try(ctx)
	assertNoError(t, err)
	state, err := limiter.Dump(ctx)
	assertNoError(t, err)
	assertFreeSlots(t, 2, state.FreeSlots)

	_, err = limiter.Try(ctx)
	assertNoError(t, err)
	state, err = limiter.Dump(ctx)
	assertNoError(t, err)
	assertFreeSlots(t, 1, state.FreeSlots)

	_, err = limiter.Try(ctx)
	assertNoError(t, err)
	state, err = limiter.Dump(ctx)
	assertNoError(t, err)
	assertFreeSlots(t, 0, state.FreeSlots)

	var res pacemaker.Result
	res, err = limiter.Try(ctx)

	assertError(t, pacemaker.ErrRateLimitExceeded, err)

	state, err = limiter.Dump(ctx)
	assertFreeSlots(t, 0, state.FreeSlots)

	time.Sleep(res.TimeToWait)

	_, err = limiter.Try(ctx)
	assertNoError(t, err)
	state, err = limiter.Dump(ctx)
	assertNoError(t, err)
	assertFreeSlots(t, 2, state.FreeSlots)

}

func TestFixedWindow_SubMinute_CapacityOne_RunOk(t *testing.T) {
	if err := db.Ping(context.Background()).Err(); err != nil {
		t.Errorf("test has failed, expected redis to be running, have error: %v", err)
	}

	opts := pacemaker.FixedWindowArgs{
		Capacity: 1,
		Rate: pacemaker.Rate{
			Amount: 3,
			Unit:   time.Second,
		},
		Clock: pacemaker.NewClock(),
		DB: pacemaker.NewFixedWindowRedisStorage(
			db,
			pacemaker.FixedWindowRedisStorageOpts{
				Prefix: "pacemaker|fixed-window|cap-1",
			},
		),
	}

	limiter := pacemaker.NewFixedWindowRateLimiter(opts)

	ctx := context.Background()

	_, err := limiter.Try(ctx)
	assertNoError(t, err)
	state, err := limiter.Dump(ctx)
	assertNoError(t, err)
	assertFreeSlots(t, 0, state.FreeSlots)

	var res pacemaker.Result
	res, err = limiter.Try(ctx)

	assertError(t, pacemaker.ErrRateLimitExceeded, err)

	state, err = limiter.Dump(ctx)
	assertFreeSlots(t, 0, state.FreeSlots)

	time.Sleep(res.TimeToWait)

	_, err = limiter.Try(ctx)
	assertNoError(t, err)
	state, err = limiter.Dump(ctx)
	assertNoError(t, err)
	assertFreeSlots(t, 0, state.FreeSlots)
	res, err = limiter.Try(ctx)

	assertError(t, pacemaker.ErrRateLimitExceeded, err)

	state, err = limiter.Dump(ctx)
	assertFreeSlots(t, 0, state.FreeSlots)

	time.Sleep(res.TimeToWait)
	state, err = limiter.Dump(ctx)
	assertFreeSlots(t, 1, state.FreeSlots)

}
