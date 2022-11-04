package main

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sonirico/pacemaker"
)

func main() {
	ctx := context.Background()

	redisOpts, err := redis.ParseURL("redis://localhost:6379/0")
	if err != nil {
		panic(err)
	}

	redisCli := redis.NewClient(redisOpts)

	rateLimit :=
		pacemaker.NewFixedWindowRateLimiter(
			pacemaker.FixedWindowArgs{
				Capacity: 1200,
				Rate: pacemaker.Rate{
					Unit:   time.Hour,
					Amount: 1,
				},
				Clock: pacemaker.NewClock(),
				DB: pacemaker.NewFixedWindowRedisStorage(
					redisCli,
					pacemaker.FixedWindowRedisStorageOpts{
						Prefix: "pacemaker",
					},
				),
			},
		)

	result, err := rateLimit.Try(ctx)
	if err != nil {
		log.Printf("error try: '%v'", err)
	}

	log.Printf("Try Result: '%v'", result)

	result, err = rateLimit.Dump(ctx)

	if err != nil {
		log.Printf("error dump: '%v'", err)
	}

	log.Printf("Dump Result: '%v'", result)
}
