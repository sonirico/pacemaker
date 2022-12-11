package pacemaker

import "errors"

var (
	ErrRateLimitExceeded         = errors.New("rate limit exceeded")
	ErrTokensGreaterThanCapacity = errors.New("tokens are greater than capacity")
	ErrCannotLoadScript          = errors.New("cannot load LUA script")
	ErrNoLastKey                 = errors.New("there is not last key")
)
