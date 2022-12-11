package pacemaker

import (
	"context"
	"errors"
	"strconv"
	"time"

	redis "github.com/go-redis/redis/v8"
)

type (
	FixedWindowRedisStorageOpts struct {
		Prefix string
	}

	FixedWindowIncArgs struct {
		Window   time.Time
		TTL      time.Duration
		Tokens   int64
		Capacity int64
	}

	FixedWindowRedisStorage struct {
		cli *redis.Client

		opts FixedWindowRedisStorageOpts

		keyGenerator func(time.Time) string
	}
)

const (
	keySep = "|"
	script = `
		local counter = tonumber(redis.call('GET', KEYS[1])) or 0
		local tokens = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])

		if counter + tokens <= capacity then
			counter = tonumber(redis.call('INCRBY', KEYS[1], tokens))
		else
			counter = counter + tokens
    end

		redis.call('PEXPIRE', KEYS[1], ARGV[3])

		return counter
	`
)

var (
	ScriptHash = Sha1Hash(script)
)

// Load will prepare this storage to be ready for usage, such as
// load into redis needed lua scripts. Calling to this method is not
// mandatory, but highly recommended.
func (s FixedWindowRedisStorage) Load(ctx context.Context) error {
	if err := s.cli.ScriptLoad(ctx, script).Err(); err != nil {
		return ErrCannotLoadScript
	}
	return nil
}

// Inc will increase, if there is room to, the rate limiting counter for the bucket
// specified by window argument.
func (s FixedWindowRedisStorage) Inc(
	ctx context.Context,
	args FixedWindowIncArgs,
) (counter int64, err error) {
	key := s.keyGenerator(args.Window)

	cmd := s.cli.EvalSha(
		ctx,
		ScriptHash,
		[]string{key},
		[]any{args.Tokens, args.Capacity, args.TTL.Milliseconds()},
	)

	if err = cmd.Err(); err != nil {
		if errIsRedisNoScript(err) {
			if err = s.cli.ScriptLoad(ctx, script).Err(); err != nil {
				err = ErrCannotLoadScript
				return
			}

			return s.Inc(ctx, args)
		}
		return
	}

	counter, err = cmd.Int64()
	return
}

func (s FixedWindowRedisStorage) Keys(ctx context.Context) (res []string, err error) {
	cmd := s.cli.Keys(ctx, s.opts.Prefix+"*")
	if err = cmd.Err(); err != nil {
		return nil, err
	}

	res, err = cmd.Result()
	return
}

func (s FixedWindowRedisStorage) LastWindow(ctx context.Context) (ts time.Time, err error) {
	var keys []string
	keys, err = s.Keys(ctx)
	if err != nil {
		return
	}

	le := len(keys)
	if le < 1 {
		err = ErrNoLastKey
		return
	}

	ts, err = LatestTsFromKeys(keys, keySep)
	return
}

func (s FixedWindowRedisStorage) Get(
	ctx context.Context,
	window time.Time,
) (counter int64, err error) {
	cmd := s.cli.Get(ctx, s.keyGenerator(window))

	if err = cmd.Err(); err != nil {
		// key does not exist
		if errors.Is(err, redis.Nil) {
			counter = 0
			err = nil
		}
		return
	}

	counter, err = cmd.Int64()
	return
}

func NewFixedWindowRedisStorage(
	cli *redis.Client,
	opts FixedWindowRedisStorageOpts,
) FixedWindowRedisStorage {
	return FixedWindowRedisStorage{
		cli:  cli,
		opts: opts,
		keyGenerator: func(t time.Time) string {
			return opts.Prefix + keySep + strconv.Itoa(int(t.UnixNano()))
		},
	}
}
