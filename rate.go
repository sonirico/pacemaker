package pacemaker

import "time"

type Rate struct {
	Amount int
	Unit   time.Duration
}

func (r Rate) Duration() time.Duration {
	return time.Duration(r.Amount) * r.Unit
}

// TruncateDuration returns, for windows smaller than a minute, the sole unit as they scape the sexagesimal counting
// mode. Otherwise, return the product of amount and unit to produce the full rate limit window.
func (r Rate) TruncateDuration() time.Duration {
	if r.Unit < time.Minute {
		return r.Unit
	}

	return r.Duration()
}
