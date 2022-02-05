# PaceMaker

Rate limit library. Currently implemented rate limits are

### Fixed window rate limit

Fixed window limits—such as 3,000 requests per hour or 10 requests per day—are easy
to state, but they are subject to spikes at the edges of the window, as
available quota resets. Consider, for example, a limit of 3,000 requests per
hour, which still allows for a spike of all 3,000 requests to be made in
the first minute of the hour, which might overwhelm the service. 

Starts counting time windows when the first request arrives.

### Fixed truncated window rate limit

Same as _Fixed Window rate limit_ but truncates the rate limit window to the rate
interval configured in order to adjust to real time intervals passing. E.g:

1. Rate limit interval is configured for new windows *every 10 seconds*
2. First request arrives at `2022-02-05 10:23:23`
3. Current rate limit window: from `2022-02-05 10:23:20` to `2022-02-05 10:23:30`


You can refer to [google architecture docs](https://cloud.google.com/architecture/rate-limiting-strategies-techniques#techniques-enforcing-rate-limits)
to read more about rate limits.


### TODO:

- Fixed window bucket variant (refill token with capacity at same rate)
- Redis storage
- Token bucket 
- Leaky bucket