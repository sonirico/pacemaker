package main

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/sonirico/pacemaker"
	"log"
	"time"
)

func main() {
	ctx := context.Background()

	redisOpts, err := redis.ParseURL("redis://localhost:6379/0")
	if err != nil {
		panic(err)
	}

	redisCli := redis.NewClient(redisOpts)

	rateLimiter := pacemaker.NewTokenFixedWindowRateLimiter(
		pacemaker.NewFixedWindowRateLimiter(pacemaker.FixedWindowArgs{
			Capacity: 1200,
			Rate: pacemaker.Rate{
				Amount: 1,
				Unit:   time.Minute,
			},
			Clock: pacemaker.NewClock(),
			DB: pacemaker.NewFixedWindowRedisStorage(redisCli, pacemaker.FixedWindowRedisStorageOpts{
				Prefix: "pacemaker",
			}),
		}),
	)

	for i := 0; i < 100; i++ {
		ttw, err := rateLimiter.Try(ctx, 100)
		log.Println(ttw, err)
		time.Sleep(time.Second)
	}
}
