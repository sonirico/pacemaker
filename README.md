# PaceMaker

Rate limit library

## Implemented rate limits:

- [Fixed Window](#fixed-window-rate-limit)
- [Fixed Window (truncated)](#fixed-truncated-window-rate-limit)
- [Token bucket & Fixed Window variant](#fixed-window-with-token-bucket-variant-token-refill-at-windows-rate)

### Fixed window rate limit

Fixed window limits—such as 3,000 requests per hour or 10 requests per day—are easy to state, but they are subject to
have spikes at the edges of the window, as available quota resets. Consider, for example, a limit of 3,000 requests per
hour, which still allows for a spike of all 3,000 requests to be made in the first minute of the hour, which might
overwhelm the service.

Starts counting time windows when the first request arrives.

[Example](./examples/fixed_window/main.go)

### Fixed truncated window rate limit

Same as _Fixed Window rate limit_ but truncates the rate limit window to the rate interval configured in order to adjust
to real time intervals passing. E.g:

1. Rate limit interval is configured for new windows *every 10 seconds*
2. First request arrives at `2022-02-05 10:23:23`
3. Current rate limit window: from `2022-02-05 10:23:20` to `2022-02-05 10:23:30`
   
[Example](./examples/fixed_window_truncated/main.go)

### Fixed window with token bucket variant (token refill at window's rate)

Works as any other fixed-window rate limiter. However, the meaning of 'capacity' of the inner fixed-window rate limiter
changes from _the total amount of requests_ that can be made to
_the total amount of tokens (points)_ that can be spent in that same window. This variant is particularly useful, for
instance, when working on the crypto game field as many crypto exchanges employ this strategy. Take for
example, [binance](https://www.binance.com/en/support/faq/360004492232).

[Example](./examples/fixed_window_token_bucket_variant/main.go)

---

You can refer to [google architecture docs](https://cloud.google.com/architecture/rate-limiting-strategies-techniques#techniques-enforcing-rate-limits)
to read more about rate limits.

## Storages

- **Memory**. Useful for non-distributed applications and testing purposes. Do not use on production unless you
  deliberately don't care about keeping rate limit state.
- **Redis**. [github.com/go-redis/redis](github.com/go-redis/redis) is employed as Redis client

### TODO:

- Token bucket rate limit
- Leaky bucket rate limit
- Composite rate limit
- Service rate limit
- Make `Try` method to return free slots
