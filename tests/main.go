package main

import (
	"context"
	"fmt"
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

	db := pacemaker.NewFixedWindowRedisStorage(redisCli, pacemaker.FixedWindowRedisStorageOpts{
		Prefix: "pacemaker",
	})

	window := time.Now().Truncate(time.Minute)

	usedTokens, err := db.Inc(ctx, pacemaker.FixedWindowIncArgs{
		Capacity: 10,
		Tokens:   6,
		TTL:      time.Minute,
		Window:   window,
	})

	if err != nil {
		panic(err)
	}

	log.Println("tokens usados", usedTokens)

	usedTokens, err = db.Inc(ctx, pacemaker.FixedWindowIncArgs{
		Capacity: 10,
		Tokens:   6,
		TTL:      time.Minute,
		Window:   window,
	})

	if err != nil {
		panic(err)
	}

	expectedTokens := int64(12)

	if expectedTokens != usedTokens {
		fmt.Printf("unexpected tokens, want %d, have %d", expectedTokens, usedTokens)
	}

	usedTokens, err = db.Get(ctx, window)

	expectedPersistedTokens := int64(6)

	if expectedPersistedTokens != usedTokens {
		fmt.Printf(
			"unexpected persisted counter, want %d, have %d",
			expectedPersistedTokens,
			usedTokens,
		)
	}
}
