package pacemaker

import "time"

type Rate struct {
	Amount int
	Unit   time.Duration
}

func (r Rate) Duration() time.Duration {
	return time.Duration(r.Amount) * r.Unit
}
