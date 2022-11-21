package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
	"github.com/sonirico/pacemaker"
	"log"
	"os"
	"testing"
	"time"
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

	if err := pool.Retry(func() error {
		db = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("localhost:%s", resource.GetPort("6379/tcp")),
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		return db.Ping(ctx).Err()
	}); err != nil {
		log.Fatalf("cannot connect to docker due to %s", err)
	}

	log.Println("redis is up and running")

	returnCode := t.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("cannot purge due to %s", err)
	}

	os.Exit(returnCode)
}

func TestFixedWindow_RunOk(t *testing.T) {
	if err := db.Ping(context.Background()).Err(); err != nil {
		t.Errorf("test has failed, expected redis to be running, have error: %v", err)
	} else {
		log.Println("redis is running ")
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
	} else {
		log.Println("redis is running ")
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
