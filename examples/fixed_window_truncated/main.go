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

	const (
		tokensPerMinute  = 1000
		tokensPerRequest = 100

		// 1 minute rate limit windows
		rateAmount = 1
		rateUnit   = time.Minute
	)

	rateLimiter :=
		pacemaker.NewFixedTruncatedWindowRateLimiter(pacemaker.FixedTruncatedWindowArgs{
			Capacity: tokensPerMinute,
			Rate: pacemaker.Rate{
				Amount: rateAmount,
				Unit:   rateUnit,
			},
			Clock: pacemaker.NewClock(),
			DB: pacemaker.NewFixedWindowRedisStorage(redisCli, pacemaker.FixedWindowRedisStorageOpts{
				Prefix: "pacemaker",
			}),
		})

	for i := 0; i < 100; i++ {
		ttw, err := rateLimiter.Try(ctx)
		log.Println(ttw, err)
		time.Sleep(time.Second)
	}
}
