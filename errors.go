package pacemaker

import "errors"

var (
	ErrRateLimitExceeded         = errors.New("rate limit exceeded")
	ErrTokensGreaterThanCapacity = errors.New("tokens are greater than capacity")
)
