package pacemaker

import (
	"context"
	"strconv"
	"time"

	redis "github.com/go-redis/redis/v8"
)

type (
	FixedWindowRedisStorageOpts struct {
		Prefix string
	}

	fixedWindowIncArgs struct {
		window time.Time
		tokens int64
		ttl    time.Duration
	}

	FixedWindowRedisStorage struct {
		cli *redis.Client

		keyGenerator func(time.Time) string
	}

	fixedWindowStorageIncArgs interface {
		TTL() time.Duration
		Window() time.Time
		Tokens() int64
	}
)

func (a fixedWindowIncArgs) Tokens() int64 {
	return a.tokens
}

func (a fixedWindowIncArgs) Window() time.Time {
	return a.window
}

func (a fixedWindowIncArgs) TTL() time.Duration {
	return a.ttl
}

func (s FixedWindowRedisStorage) Inc(ctx context.Context, args fixedWindowStorageIncArgs) (counter int64, err error) {
	var incrCmd *redis.IntCmd

	_, err = s.cli.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		key := s.keyGenerator(args.Window())
		incrCmd = pipe.IncrBy(ctx, key, args.Tokens())
		return pipe.PExpire(ctx, key, args.TTL()).Err()
	})

	if err != nil {
		return
	}

	counter = incrCmd.Val()
	err = incrCmd.Err()

	return
}

func (s FixedWindowRedisStorage) Get(ctx context.Context, window time.Time) (counter int64, err error) {
	cmd := s.cli.Get(ctx, s.keyGenerator(window))

	err = cmd.Err()

	if err != nil {
		return
	}

	counter, err = cmd.Int64()
	return
}

func NewFixedWindowRedisStorage(cli *redis.Client, opts FixedWindowRedisStorageOpts) FixedWindowRedisStorage {
	return FixedWindowRedisStorage{
		cli: cli,
		keyGenerator: func(t time.Time) string {
			return opts.Prefix + "|" + strconv.Itoa(int(t.UnixNano()))
		},
	}
}
